package message

import (
	"time"

	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type Http struct {
	Received chan string  // received messages from web
	ToSend   chan Message // messages to send to web
}

type Req struct {
	Message string `json:"message" form:"message"`
}

// GetHandle handles POST /api/admin/message/get
//
// @Summary		Get Message
// @Description	Get a message from the server
// @Tags			Admin
// @Accept			json
// @Produce		json
// @Success		200	{object}	common.jsonResult{data=Message}	"Message data"
// @Failure		404	{object}	common.jsonResult{data=string}	"Not Found"
// @Router			/api/admin/message/get [post]
func (p *Http) GetHandle(c *gin.Context) {
	select {
	case message := <-p.ToSend:
		common.SuccessResp(c, message)
	default:
		common.ErrorStrResp(c, "no message", 404)
	}
}

// SendHandle handles POST /api/admin/message/send
//
// @Summary		Send Message
// @Description	Send a message to the server
// @Tags			Admin
// @Accept			json
// @Produce		json
// @Param			message	body		Req					true	"Message Data"
// @Success		200		{object}	common.jsonResult{data=string}	"Success message"
// @Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
// @Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
// @Router			/api/admin/message/send [post]
func (p *Http) SendHandle(c *gin.Context) {
	var req Req
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	select {
	case p.Received <- req.Message:
		common.SuccessResp(c)
	default:
		common.ErrorStrResp(c, "nowhere needed", 500)
	}
}

func (p *Http) Send(message Message) error {
	select {
	case p.ToSend <- message:
		return nil
	default:
		return errors.New("send failed")
	}
}

func (p *Http) Receive() (string, error) {
	select {
	case message := <-p.Received:
		return message, nil
	default:
		return "", errors.New("receive failed")
	}
}

func (p *Http) WaitSend(message Message, d int) error {
	select {
	case p.ToSend <- message:
		return nil
	case <-time.After(time.Duration(d) * time.Second):
		return errors.New("send timeout")
	}
}

func (p *Http) WaitReceive(d int) (string, error) {
	select {
	case message := <-p.Received:
		return message, nil
	case <-time.After(time.Duration(d) * time.Second):
		return "", errors.New("receive timeout")
	}
}

var HttpInstance = &Http{
	Received: make(chan string),
	ToSend:   make(chan Message),
}
