package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func fetchMembers(c *gin.Context) {

	_, pageSize, cookie := findPageControl(c)

	sr, err := org.Members(pageSize, cookie)
	if err != nil {
		sendError(c, err)
		return
	}

	sendResult(c, sr)
}

func mapToMemberInfo(body map[string]interface{}) map[string][]string {

	memberInfo := make(map[string][]string, 0)

	if name, ok := body[`name`].(string); ok {
		memberInfo[`name`] = []string{name}
	}
	if cn, ok := body[`cn`].(string); ok {
		memberInfo[`cn`] = []string{cn}
	}
	if email, ok := body[`email`].(string); ok {
		memberInfo[`email`] = []string{email}
	}
	if gender, ok := body[`gender`].(string); ok {
		memberInfo[`gender`] = []string{gender}
	}
	if priority, ok := body[`priority`].(string); ok {
		memberInfo[`priority`] = []string{priority}
	}
	if telephoneNumber, ok := body[`telephoneNumber`].(string); ok {
		memberInfo[`telephoneNumber`] = []string{telephoneNumber}
	}
	if title, ok := body[`title`].(string); ok {
		memberInfo[`title`] = []string{title}
	}
	if unitIDs, ok := body[`unitID`].([]interface{}); ok {
		var ids []string
		for _, id := range unitIDs {
			ids = append(ids, id.(string))
		}
		memberInfo[`unitID`] = ids
	}
	if rbacType, ok := body[`rbacType`].(string); ok {
		memberInfo[`rbacType`] = []string{rbacType}
	}
	if rbacRoles, ok := body[`rbacRole`].([]interface{}); ok {
		var roles []string
		for _, r := range rbacRoles {
			roles = append(roles, r.(string))
		}
		memberInfo[`rbacRole`] = roles
	}

	return memberInfo
}

func createMember(c *gin.Context) {

	var body map[string]interface{}
	err := c.BindJSON(&body) // 会发送错误信息
	if err != nil {
		return
	}

	memberInfo := mapToMemberInfo(body)

	m := md5.New()
	m.Write([]byte(`123456`))
	pwd := m.Sum(nil)

	memberInfo[`userPassword`] = []string{fmt.Sprintf(`{MD5}%s`, hex.EncodeToString(pwd))}

	id, err := org.AddMember(memberInfo)
	if err != nil {
		sendError(c, err)
		return
	}

	thirdPwd := newPassword()

	// 去环信注册
	err = em.RegisterSignelUser(id, thirdPwd)
	if err != nil {
		log.WithField(`resource`, `member`).Warn(fmt.Sprintf(`register user failed %v`, err))
	}
	// 将用户名和密码更新到ldapserver
	err = org.ModifyMember(id, map[string][]string{
		`thirdAccount`:  []string{id},
		`thirdPassword`: []string{thirdPwd},
		`labeledURI`:    []string{id}, // 顺便在同一个请求中添加默认头像 如果创建用户成功，则会在后台自动生成默认头像
	})
	if err != nil {
		log.WithField(`resource`, `memeber`).Warn(fmt.Sprintf(`modify member third account info occured error %v`, err))
	}

	// 生成默认头像
	go func(n, id string) {
		a, err := avatarGenerator.Gen(n)
		if err != nil {
			log.WithField(`resource`, `generate avatar`).Warn(err)
			return
		}
		go func(string, string) {
			_, err = uploadFileToQiNiu(a, id)
			if err != nil {
				log.WithField(`resource`, `upload avatar`).Warn(err)
				return
			}
		}(a, id)
	}(memberInfo[`name`][0], id)

	c.JSON(http.StatusOK, map[string]string{`id`: id})
}

func editMember(c *gin.Context) {
	var body map[string]interface{}
	err := c.BindJSON(&body) // 会发送错误信息
	if err != nil {
		return
	}

	memberInfo := mapToMemberInfo(body)

	id := c.Param(`id`)
	err = org.ModifyMember(id, memberInfo)
	if err != nil {
		sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, map[string]string{`id`: id})
}

func memberByID(c *gin.Context) {
	ms, err := org.MemberByID(c.Param(`id`), true, false)
	if err != nil {
		sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, ms)
}

func delMember(c *gin.Context) {

	id := c.Param(`id`)

	err := org.DelMember(id)
	if err != nil {
		sendError(c, err)
		return
	}
	// 去环信删除
	err = em.DeleteUser(id)
	if err != nil {
		log.WithField(`resource`, `member`).Warn(fmt.Sprintf(`delete user[id:%v] failed %v`, id, err))
	}

	c.JSON(http.StatusOK, map[string]string{`id`: id})
}
