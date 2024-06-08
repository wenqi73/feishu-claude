package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"start-feishubot/utils"
	"strings"
	"time"
)

func setDefaultClaudePrompt(msg []utils.Messages) []utils.Messages {
	if !hasSystemRole(msg) {
		msg = append(msg, utils.Messages{
			Role: "system", Content: "You are Claude, " +
				"Answer in user's language as concisely as" +
				" possible. Knowledge cutoff: 20230601 " +
				"Current date" + time.Now().Format("20060102"),
		})
	}
	return msg
}

type ClaudeMessageAction struct { /*消息*/
}

func (*ClaudeMessageAction) Execute(a *ActionInfo) bool {
	if a.handler.config.StreamMode {
		return true
	}
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	msg = setDefaultClaudePrompt(msg)
	msg = append(msg, utils.Messages{
		Role: "user", Content: a.info.qParsed,
	})

	// get ai mode as temperature
	fmt.Println("msg: ", msg)
	completions, err := a.handler.claude.Completions(msg)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	msg = append(msg, completions)
	a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
	//if new topic
	if len(msg) == 3 {
		//fmt.Println("new topic", msg[1].Content)
		sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId,
			completions.Content)
		return false
	}
	if len(msg) != 3 {
		sendOldTopicCard(*a.ctx, a.info.sessionId, a.info.msgId,
			completions.Content)
		return false
	}
	err = replyMsg(*a.ctx, completions.Content, a.info.msgId)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	return true
}

type StreamClaudeMessageAction struct { /*消息*/
}

func (m *StreamClaudeMessageAction) Execute(a *ActionInfo) bool {
	if !a.handler.config.StreamMode {
		return true
	}
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	// 如果没有提示词，默认模拟ChatGPT
	msg = setDefaultClaudePrompt(msg)
	msg = append(msg, utils.Messages{
		Role: "user", Content: a.info.qParsed,
	})
	//if new topic
	var ifNewTopic bool
	if len(msg) <= 3 {
		ifNewTopic = true
	} else {
		ifNewTopic = false
	}

	cardId, err2 := sendOnProcess(a, ifNewTopic)
	if err2 != nil {
		return false
	}

	answer := ""
	chatResponseStream := make(chan string)
	done := make(chan struct{}) // 添加 done 信号，保证 goroutine 正确退出
	noContentTimeout := time.AfterFunc(10*time.Second, func() {
		log.Println("no content timeout")
		close(done)
		err := updateFinalCard(*a.ctx, "请求超时", cardId, ifNewTopic)
		if err != nil {
			return
		}
		return
	})
	defer noContentTimeout.Stop()

	go func() {
		defer func() {
			if err := recover(); err != nil {
				err := updateFinalCard(*a.ctx, "聊天失败", cardId, ifNewTopic)
				if err != nil {
					return
				}
			}
		}()

		//log.Printf("UserId: %s , Request: %s", a.info.userId, msg)
		aiMode := a.handler.sessionCache.GetAIMode(*a.info.sessionId)
		//fmt.Println("msg: ", msg)
		//fmt.Println("aiMode: ", aiMode)
		if err := a.handler.gpt.StreamChat(*a.ctx, msg, aiMode,
			chatResponseStream); err != nil {
			err := updateFinalCard(*a.ctx, "聊天失败", cardId, ifNewTopic)
			if err != nil {
				return
			}
			close(done) // 关闭 done 信号
		}

		close(done) // 关闭 done 信号
	}()
	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop() // 注意在函数结束时停止 ticker
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				err := updateTextCard(*a.ctx, answer, cardId, ifNewTopic)
				if err != nil {
					return
				}
			}
		}
	}()
	for {
		select {
		case res, ok := <-chatResponseStream:
			if !ok {
				return false
			}
			noContentTimeout.Stop()
			answer += res
			//pp.Println("answer", answer)
		case <-done: // 添加 done 信号的处理
			err := updateFinalCard(*a.ctx, answer, cardId, ifNewTopic)
			if err != nil {
				return false
			}
			ticker.Stop()
			msg := append(msg, utils.Messages{
				Role: "assistant", Content: answer,
			})
			a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
			close(chatResponseStream)
			log.Printf("\n\n\n")
			jsonByteArray, err := json.Marshal(msg)
			if err != nil {
				log.Println(err)
			}
			jsonStr := strings.ReplaceAll(string(jsonByteArray), "\\n", "")
			jsonStr = strings.ReplaceAll(jsonStr, "\n", "")
			log.Printf("\n\n\n")
			return false
		}
	}
}
