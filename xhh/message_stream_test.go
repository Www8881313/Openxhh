package xhh

import (
	"openxhh/db"
	"testing"
)

func TestFindTrackedBotCommentUsesCommentID(t *testing.T) {
	comments := []CommentInfo{
		{CommentID: 10, UserID: 1, Text: "其他评论"},
		{CommentID: 20, UserID: 2, Text: "机器人回复被小黑盒改写"},
	}
	outbound := db.OutboundMessage{CommentID: 20, Text: "原始机器人回复"}

	got := findTrackedBotComment(comments, outbound)
	if got == nil || got.CommentID != 20 {
		t.Fatalf("findTrackedBotComment = %#v, want comment 20", got)
	}
}

func TestFindTrackedBotCommentUsesNormalizedText(t *testing.T) {
	comments := []CommentInfo{
		{CommentID: 30, UserID: 1, Text: "机器人 回复 [cube_喜欢]"},
	}
	outbound := db.OutboundMessage{Text: "机器人回复"}

	got := findTrackedBotComment(comments, outbound)
	if got == nil || got.CommentID != 30 {
		t.Fatalf("findTrackedBotComment = %#v, want comment 30", got)
	}
}

func TestFindTrackedBotCommentUsesBotIdentityForReply(t *testing.T) {
	oldHeyBoxID := Info.HeyBoxId
	Info.HeyBoxId = "42"
	t.Cleanup(func() { Info.HeyBoxId = oldHeyBoxID })

	comments := []CommentInfo{
		{CommentID: 40, UserID: 7, ReplyID: 10, Text: "普通用户"},
		{CommentID: 41, UserID: 42, ReplyID: 10, Text: "机器人回复被服务端改写"},
	}
	outbound := db.OutboundMessage{RootCommentID: 1, ReplyCommentID: 10, Text: "原始机器人回复"}

	got := findTrackedBotComment(comments, outbound)
	if got == nil || got.CommentID != 41 {
		t.Fatalf("findTrackedBotComment = %#v, want comment 41", got)
	}
}

func TestFindTrackedBotCommentDoesNotGuessWithoutAnchor(t *testing.T) {
	oldHeyBoxID := Info.HeyBoxId
	Info.HeyBoxId = "42"
	t.Cleanup(func() { Info.HeyBoxId = oldHeyBoxID })

	comments := []CommentInfo{{CommentID: 50, UserID: 42, Text: "机器人回复被服务端改写"}}
	outbound := db.OutboundMessage{Text: "原始机器人回复"}

	if got := findTrackedBotComment(comments, outbound); got != nil {
		t.Fatalf("findTrackedBotComment = %#v, want nil", got)
	}
}

func TestShouldSaveTrackedInboundForReplyToBot(t *testing.T) {
	oldHeyBoxID := Info.HeyBoxId
	Info.HeyBoxId = "42"
	t.Cleanup(func() { Info.HeyBoxId = oldHeyBoxID })

	comment := CommentInfo{CommentID: 61, UserID: 7, ReplyID: 60, Text: "没 @，但回复机器人"}
	outbound := db.OutboundMessage{Text: "机器人回复"}

	if !shouldSaveTrackedInbound(comment, 10, 60, outbound) {
		t.Fatal("shouldSaveTrackedInbound should save reply to bot")
	}
}

func TestShouldSaveTrackedInboundForBotFloorComment(t *testing.T) {
	oldHeyBoxID := Info.HeyBoxId
	Info.HeyBoxId = "42"
	t.Cleanup(func() { Info.HeyBoxId = oldHeyBoxID })

	comment := CommentInfo{CommentID: 71, UserID: 7, Text: "机器人楼层下的新评论"}
	outbound := db.OutboundMessage{Text: "机器人顶级评论"}

	if !shouldSaveTrackedInbound(comment, 70, 70, outbound) {
		t.Fatal("shouldSaveTrackedInbound should save comment on bot floor")
	}
}

func TestShouldSaveTrackedInboundSkipsBotAndUnresolvedBotComment(t *testing.T) {
	oldHeyBoxID := Info.HeyBoxId
	Info.HeyBoxId = "42"
	t.Cleanup(func() { Info.HeyBoxId = oldHeyBoxID })

	outbound := db.OutboundMessage{Text: "机器人回复"}
	if shouldSaveTrackedInbound(CommentInfo{CommentID: 81, UserID: 42, ReplyID: 80, Text: "机器人自己"}, 10, 80, outbound) {
		t.Fatal("should skip bot's own comment")
	}
	if shouldSaveTrackedInbound(CommentInfo{CommentID: 82, UserID: 7, ReplyID: 80, Text: "用户回复"}, 10, 0, outbound) {
		t.Fatal("should skip when bot comment is unresolved")
	}
}
