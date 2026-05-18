package xhh

import "testing"

func TestParseImageCommand(t *testing.T) {
	tests := []struct {
		name              string
		text              string
		wantOK            bool
		wantPrompt        string
		wantPostContext   bool
		wantCommentCtx    bool
		wantImageInput    bool
		wantMentionTarget string
	}{
		{
			name:       "strong trigger",
			text:       "@机器人 生图 一只赛博朋克猫",
			wantOK:     true,
			wantPrompt: "一只赛博朋克猫",
		},
		{
			name:       "weak generate one image",
			text:       "@机器人 生成一张机械朋克猫",
			wantOK:     true,
			wantPrompt: "机械朋克猫",
		},
		{
			name:       "weak draw image",
			text:       "@机器人 帮我画小黑盒头像",
			wantOK:     true,
			wantPrompt: "小黑盒头像",
		},
		{
			name:       "cleanup mention control",
			text:       "@机器人 生图 一张小菲的画像，并艾特她查看",
			wantOK:     true,
			wantPrompt: "一张小菲的画像",
			wantMentionTarget: "小菲",
		},
		{
			name:              "explicit mention target",
			text:              "@机器人 生图 一只猫，顺便艾特小明看看",
			wantOK:            true,
			wantPrompt:        "一只猫",
			wantMentionTarget: "小明",
		},
		{
			name:            "post context",
			text:            "@机器人 根据正文生成一张梗图",
			wantOK:          true,
			wantPrompt:      "梗图",
			wantPostContext: true,
		},
		{
			name:           "comment context",
			text:           "@机器人 根据评论区内容生成一张海报",
			wantOK:         true,
			wantPrompt:     "海报",
			wantCommentCtx: true,
		},
		{
			name:           "image input flag",
			text:           "@机器人 参考这张图生成一张赛博朋克头像",
			wantOK:         true,
			wantPrompt:     "赛博朋克头像",
			wantImageInput: true,
		},
		{
			name:   "not text generation",
			text:   "@机器人 生成一段回复",
			wantOK: false,
		},
		{
			name:   "not empty prompt",
			text:   "@机器人 生图",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseImageCommand(tt.text)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v, command=%+v", ok, tt.wantOK, got)
			}
			if !ok {
				return
			}
			if got.Prompt != tt.wantPrompt {
				t.Fatalf("Prompt = %q, want %q", got.Prompt, tt.wantPrompt)
			}
			if got.UsePostContext != tt.wantPostContext {
				t.Fatalf("UsePostContext = %v, want %v", got.UsePostContext, tt.wantPostContext)
			}
			if got.UseCommentContext != tt.wantCommentCtx {
				t.Fatalf("UseCommentContext = %v, want %v", got.UseCommentContext, tt.wantCommentCtx)
			}
			if got.UseImageInput != tt.wantImageInput {
				t.Fatalf("UseImageInput = %v, want %v", got.UseImageInput, tt.wantImageInput)
			}
			if got.MentionTargetText != tt.wantMentionTarget {
				t.Fatalf("MentionTargetText = %q, want %q", got.MentionTargetText, tt.wantMentionTarget)
			}
		})
	}
}

func TestExtractImagePromptCompatibility(t *testing.T) {
	prompt, ok := ExtractImagePrompt("@机器人 生成一张机械朋克猫")
	if !ok {
		t.Fatal("ExtractImagePrompt returned false")
	}
	if prompt != "机械朋克猫" {
		t.Fatalf("prompt = %q, want %q", prompt, "机械朋克猫")
	}
}
