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

type APIError struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id"`
}

func RESTHandler[TReq, TResp any](handler func(context.Context, *TReq) (TResp, error)) func(*gin.Context) {
	return RESTHandlerWithUriParams(func(ctx context.Context, req *TReq, uri *Empty) (TResp, error) {
		return handler(ctx, req)
	})
}

func RESTHandlerWithUriParams[TReq, TResp, TUri any](handler func(context.Context, *TReq, *TUri) (TResp, error)) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logrus.WithContext(c)
		uriReq := new(TUri)
		requestID := requestid.Get(c)
		err := c.ShouldBindUri(uriReq)
		if err != nil {
			log.WithError(err).Error()
			c.JSON(http.StatusBadRequest, APIError{
				Error:     err.Error(),
				RequestID: requestID,
			})
			return
		}
		req := new(TReq)
		err = c.ShouldBind(req)
		if err != nil {
			log.WithError(err).Error()
			c.JSON(http.StatusBadRequest, APIError{
				Error:     err.Error(),
				RequestID: requestID,
			})
			return
		}
		resp, err := handler(c, req, uriReq)
		if err != nil {
			log.WithError(err).Error()
			c.JSON(http.StatusBadRequest, &APIError{
				Error:     err.Error(),
				RequestID: requestID,
			})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Error   string `json:"error,omitempty"`
}

func APIHandler[TReq, TResp any](handler func(context.Context, *TReq) (TResp, error)) func(*gin.Context) {
	return APIHandlerWithUriParams(func(ctx context.Context, req *TReq, uri *Empty) (TResp, error) {
		return handler(ctx, req)
	})
}

func APIHandlerWithUriParams[TReq, TResp, TUri any](handler func(context.Context, *TReq, *TUri) (TResp, error)) func(*gin.Context) {
	return func(c *gin.Context) {
		log := logrus.WithContext(c)
		uriReq := new(TUri)
		err := c.ShouldBindUri(uriReq)
		if err != nil {
			log.WithError(err).Error()
			c.JSON(http.StatusBadRequest, APIResponse{
				Error: err.Error(),
			})
			return
		}
		req := new(TReq)
		err = c.ShouldBind(req)
		if err != nil {
			log.WithError(err).Error()
			c.JSON(http.StatusBadRequest, APIResponse{
				Error: err.Error(),
			})
			return
		}
		resp, err := handler(c, req, uriReq)
		if err != nil {
			log.WithError(err).Error()
			c.JSON(http.StatusOK, &APIResponse{
				Error: err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, &APIResponse{
			Success: true,
			Data:    resp,
		})
	}
}
