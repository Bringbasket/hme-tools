package systemupdate

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gokeep/server/internal/common"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register exposes version status to configuration readers and mutating
// operations to configuration editors.
func (handler *Handler) Register(group *gin.RouterGroup, read, edit gin.HandlerFunc) {
	group.GET("/version", read, handler.status)
	group.POST("/version/check", edit, handler.check)
	group.POST("/version/update", edit, handler.update)
}

func (handler *Handler) status(c *gin.Context) {
	c.JSON(http.StatusOK, common.OK(handler.service.Status()))
}

func (handler *Handler) check(c *gin.Context) {
	status, err := handler.service.Check()
	if errors.Is(err, ErrInProgress) {
		c.JSON(http.StatusConflict, common.Fail(http.StatusConflict, "已有版本任务正在运行"))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(http.StatusInternalServerError, "版本检查请求暂时无法保存"))
		return
	}
	c.JSON(http.StatusAccepted, common.OK(status))
}

func (handler *Handler) update(c *gin.Context) {
	status, err := handler.service.Request()
	switch {
	case errors.Is(err, ErrInProgress):
		c.JSON(http.StatusConflict, common.Fail(http.StatusConflict, "已有更新任务正在运行"))
	case errors.Is(err, ErrCheckRequired):
		c.JSON(http.StatusConflict, common.Fail(http.StatusConflict, "请先检查是否有可用更新"))
	case errors.Is(err, ErrNoUpdateAvailable):
		c.JSON(http.StatusConflict, common.Fail(http.StatusConflict, "当前已经是最新版本"))
	case err != nil:
		c.JSON(http.StatusInternalServerError, common.Fail(http.StatusInternalServerError, "更新请求暂时无法保存"))
	default:
		c.JSON(http.StatusAccepted, common.OK(status))
	}
}
