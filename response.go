package ginx

import (
	"context"
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Empty struct {
}

type CommonResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Error   string `json:"error"`
	Req     string `json:"req"`
}

func CommonHandler[TReq, TResp any](handler func(context.Context, *TReq) (TResp, error)) func(*gin.Context) {
	return CommonHandlerWithUriParams(func(ctx context.Context, req *TReq, uri *Empty) (TResp, error) {
		return handler(ctx, req)
	})
}

func CommonHandlerWithUriParams[TReq, TResp, TUri any](handler func(context.Context, *TReq, *TUri) (TResp, error)) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logrus.WithContext(c)
		requestID := requestid.Get(c)
		uriReq := new(TUri)
		err := c.ShouldBindUri(uriReq)
		if err != nil {
			log.WithError(err).Error()
			c.JSON(http.StatusBadRequest, CommonResponse{
				Error: err.Error(),
				Req:   requestID,
			})
			return
		}
		req := new(TReq)
		err = c.ShouldBind(req)
		if err != nil {
			log.WithError(err).Error()
			c.JSON(http.StatusBadRequest, CommonResponse{
				Error: err.Error(),
				Req:   requestID,
			})
			return
		}
		resp, err := handler(c, req, uriReq)
		if err != nil {
			log.WithError(err).Error()
			c.JSON(http.StatusOK, &CommonResponse{
				Error: err.Error(),
				Req:   requestID,
			})
			return
		}
		c.JSON(http.StatusOK, &CommonResponse{
			Success: true,
			Data:    resp,
			Req:     requestID,
		})
	}
}
