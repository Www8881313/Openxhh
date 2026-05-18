package xhh

import (
	"encoding/json"
	"testing"
)

func TestMsgUnmarshalAtPost(t *testing.T) {
	data := []byte(`{
		"message_id": 1001,
		"message_type": 16,
		"user_a": {"userid": "89055874", "username": "小猫娘喵喵"},
		"link": {"linkid": 181099114, "description": "@机器人 生成一张机械朋克猫"}
	}`)

	var msg Msg
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !msg.IsPost {
		t.Fatal("IsPost should be true")
	}
	if msg.CommentID != -1 || msg.RootCommentID != -1 {
		t.Fatalf("comment ids = (%d,%d), want (-1,-1)", msg.CommentID, msg.RootCommentID)
	}
	if msg.LinkID != 181099114 {
		t.Fatalf("LinkID = %d", msg.LinkID)
	}
	if msg.UserID != 89055874 {
		t.Fatalf("UserID = %d", msg.UserID)
	}
	if msg.UserName != "小猫娘喵喵" {
		t.Fatalf("UserName = %q", msg.UserName)
	}
	if msg.CommentText != "@机器人 生成一张机械朋克猫" {
		t.Fatalf("CommentText = %q", msg.CommentText)
	}
}

func TestMsgUnmarshalAtComment(t *testing.T) {
	data := []byte(`{
		"message_id": 1002,
		"message_type": 17,
		"userid_a": 89055874,
		"comment_a_id": 867937626,
		"root_comment_id": 867937000,
		"linkid": 181099114,
		"comment_a_text": "@机器人 生图 一只猫"
	}`)

	var msg Msg
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if msg.IsPost {
		t.Fatal("IsPost should be false")
	}
	if msg.CommentID != 867937626 || msg.RootCommentID != 867937000 {
		t.Fatalf("comment ids = (%d,%d)", msg.CommentID, msg.RootCommentID)
	}
	if msg.LinkID != 181099114 || msg.UserID != 89055874 {
		t.Fatalf("LinkID/UserID = %d/%d", msg.LinkID, msg.UserID)
	}
}

func TestExtractExplicitMentionTargetConversationCommands(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{text: "@机器人 向小明打个招呼", want: "小明"},
		{text: "@机器人 和小明说说", want: "小明"},
		{text: "@机器人 跟小红聊两句", want: "小红"},
		{text: "@机器人 对@阿伟说一下", want: "阿伟"},
		{text: "@机器人 咬小明一口", want: "小明"},
		{text: "@机器人 反驳小明的观点", want: "小明"},
		{text: "@机器人 问问小周怎么看", want: "小周"},
		{text: "@机器人 生成一张小菲的画像，并艾特小菲来看[cube_喜欢]", want: "小菲"},
		{text: "@机器人 生图 一只猫，顺便艾特小明看看[cube_点赞]", want: "小明"},
		{text: "@机器人 反驳他", want: ""},
	}

	for _, tt := range cases {
		if got := extractExplicitMentionTarget(tt.text); got != tt.want {
			t.Fatalf("extractExplicitMentionTarget(%q) = %q, want %q", tt.text, got, tt.want)
		}
	}
}
