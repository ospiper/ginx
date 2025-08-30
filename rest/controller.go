package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ospiper/ginx/dbx"
)

type IDQueryInPath struct {
	ID int64 `uri:"id" binding:"required"`
}

type PagedResults[T any] struct {
	Records []*T  `json:"records"`
	Total   int64 `json:"total"`
}

type SimpleRestQuery struct {
	Filter string `form:"filter"`
	Sort   string `form:"sort"`
	Range  string `form:"range"`
	Embed  string `form:"embed"`
}

var regRange = regexp.MustCompile(`^\[(\d+)[-,]\s*(\d+)]$`)

// BuildSimpleRestConditions ?sort=["title","ASC"]&range=[0, 24]&filter={"title":"bar"}
func BuildSimpleRestConditions(c *gin.Context) (*FindConditions, error) {
	req := new(SimpleRestQuery)
	err := c.ShouldBindQuery(req)
	if err != nil {
		return nil, err
	}
	// embed
	var embed []string
	if req.Embed != "" {
		err = json.Unmarshal([]byte(req.Embed), &embed)
		if err != nil {
			return nil, err
		}
	}

	// range [start,end]
	ranges := regRange.FindStringSubmatch(req.Range)
	page := &Range{}
	if len(ranges) > 1 {
		start, err := strconv.ParseInt(ranges[1], 10, 32)
		if err != nil {
			return nil, err
		}
		end, err := strconv.ParseInt(ranges[2], 10, 32)
		if err != nil {
			return nil, err
		}
		page.Start, page.End = int(start), int(end)
	} else {
		page.Start = 0
		page.End = 25
	}

	// sort
	var sort []string
	if req.Sort != "" {
		err = json.Unmarshal([]byte(req.Sort), &sort)
		if err != nil {
			return nil, err
		}
	} else {
		sort = []string{"id", "asc"}
	}
	if len(sort)%2 != 0 {
		return nil, errors.New("sort must be pairs")
	}
	var orders []Order
	for i := 0; i < len(sort); i += 2 {
		field, order := sort[i], strings.ToLower(sort[i+1])
		orders = append(orders, Order{
			Column: field,
			Desc:   order == "desc",
		})
	}

	// filter
	filters, err := buildFilters(req.Filter)
	if err != nil {
		return nil, err
	}
	return &FindConditions{
		Filters:    filters,
		Orders:     orders,
		Preloads:   embed,
		Pagination: page,
	}, nil
}

type ResourceController[T dbx.ModelStruct[T]] struct {
	Name     string
	Provider Provider[T]
	Group    *gin.RouterGroup
}

func (c *ResourceController[T]) Register() {
	RegisterResourceController(c.Group, c.Provider)
}

func NestedController[TBase dbx.ModelStruct[TBase], TNest dbx.ModelStruct[TNest]](baseController *ResourceController[TBase], nestController *ResourceController[TNest], name string) {
	nestBaseGroup := baseController.Group.Group(":id").Group(strings.ToLower(name))
	// controllers coping with nested (foreign key restraint) structures
	// should not be nested in the API, it should directly be /tags/:id

	// for OneToMany relations, is it necessary to keep interfaces like GET /tags and POST /tags
	// to get all or to create a new tag even it might be meaningless?
	nestBaseGroup.GET("", func(c *gin.Context) { // /drives/:id/tags
		params := &IDQueryInPath{}
		err := c.ShouldBindUri(params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cond, err := BuildSimpleRestConditions(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var parentModel TBase
		p := parentModel.NewWithID(params.ID)
		records, err := nestController.Provider.FindAssoc(c, &p, name, cond)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cnt, err := nestController.Provider.CountAssoc(c, &p, name, cond.Filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		code, hd := PaginationHeader(cond.Pagination, cnt)
		if hd != "" {
			c.Header("Content-Range", hd)
		}
		c.JSON(code, records)
	})
}

func RegisterResourceController[T dbx.ModelStruct[T]](base *gin.RouterGroup, provider Provider[T]) *ResourceController[T] {
	base.GET("", func(c *gin.Context) { // /drives
		cond, err := BuildSimpleRestConditions(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		records, err := provider.Find(c, cond)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cnt, err := provider.Count(c, cond.Filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		code, hd := PaginationHeader(cond.Pagination, cnt)
		if hd != "" {
			c.Header("Content-Range", hd)
		}
		c.JSON(code, records)
	})
	base.POST("", func(c *gin.Context) { // /drives
		var data T
		err := c.ShouldBind(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err = provider.Insert(c, &data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, data)
	})
	idGroup := base.Group(":id")
	idGroup.GET("", func(c *gin.Context) { // /drives/:id
		params := &IDQueryInPath{}
		err := c.ShouldBindUri(params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ret, err := provider.FindOne(c, params.ID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ret)
	})
	idGroup.PUT("", func(c *gin.Context) { // /drives/:id
		params := &IDQueryInPath{}
		err := c.ShouldBindUri(params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var data T
		err = c.ShouldBind(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err = provider.Update(c, params.ID, &data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, data)
	})
	idGroup.DELETE("", func(c *gin.Context) { // /drives/:id
		params := &IDQueryInPath{}
		err := c.ShouldBindUri(params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err = provider.Delete(c, params.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNoContent, nil)
	})
	return &ResourceController[T]{
		Name:     "resource",
		Provider: provider,
		Group:    base,
	}
}
