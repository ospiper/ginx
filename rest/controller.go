package sentinel

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/ospiper/ginx/dbx"
)

type IDQueryInPath struct {
	ID int64 `uri:"id" binding:"required"`
}

type PagedResults[T any] struct {
	Records []*T  `json:"records"`
	Total   int64 `json:"total"`
	*Pagination
}

func buildOrders(q url.Values) ([]Order, error) {
	if !q.Has("order") {
		return []Order{
			{Column: "id", Desc: false},
		}, nil
	}
	orders := q["order"]
	desc := q["desc"]
	if len(desc) > 0 && len(desc) != len(orders) {
		return nil, errors.New("length of desc must correspond with order")
	}
	ret := make([]Order, 0, len(orders))
	for i, o := range orders {
		d := false
		var err error
		if len(desc) > 0 {
			d, err = strconv.ParseBool(desc[i])
			if err != nil {
				return nil, err
			}
		}
		ret = append(ret, Order{Column: o, Desc: d})
	}
	return ret, nil
}

// BuildConditions ?page=1&limit=100&order=id&desc=true&order=created_at&desc=true&abs_path[Regex]=%2FVolumes.*&name=TestDrive
// Available conditions:
// Eq(optional), NotEq
// Gt, Lt, Gte, Lte
// Like, NotLike
// Between, NotBetween
// In, NotIn
// Regex
// Ts ts_query
func BuildConditions(c *gin.Context) (*FindConditions, error) {
	pagination := new(Pagination)
	err := c.ShouldBindQuery(pagination)
	if err != nil {
		return nil, err
	}
	queries := c.Request.URL.Query()
	queries.Del("page")
	queries.Del("limit")
	orders, err := buildOrders(queries)
	if err != nil {
		return nil, err
	}
	queries.Del("order")
	queries.Del("desc")
	matchPattern, _ := regexp.Compile(`^([a-zA-Z]\w*)(\[(\w+)])?$`)
	filters := make(map[string]map[string][]string) // {id: {gt: 1, lt: 2}}
	for k, v := range queries {
		matches := matchPattern.FindStringSubmatch(k)
		if len(matches) == 0 {
			continue
		}
		field := matches[1]
		verb := matches[3]
		if verb == "" {
			verb = "eq"
		}
		if _, ok := filters[field]; !ok {
			filters[field] = make(map[string][]string)
		}
		filters[field][verb] = v
	}
	filter := make(map[string]*Filter, len(filters))
	for k, v := range filters {
		_f := new(Filter)
		err := mapstructure.Decode(v, _f)
		if err != nil {
			return nil, err
		}
		filter[k] = _f
	}
	ret := &FindConditions{
		Filters:    filter,
		Orders:     orders,
		Pagination: pagination,
	}
	return ret, nil
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
		cond, err := BuildConditions(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var parentModel TBase
		p := parentModel.ID(params.ID)
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
		c.JSON(http.StatusOK, &PagedResults[TNest]{
			Records:    records,
			Total:      cnt,
			Pagination: cond.Pagination,
		})
	})
}

func RegisterResourceController[T dbx.ModelStruct[T]](base *gin.RouterGroup, provider Provider[T]) *ResourceController[T] {
	base.GET("", func(c *gin.Context) { // /drives
		cond, err := BuildConditions(c)
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
		c.JSON(http.StatusOK, &PagedResults[T]{
			Records:    records,
			Total:      cnt,
			Pagination: cond.Pagination,
		})
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
