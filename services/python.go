package services

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/crawlab-team/crawlab-core/controllers"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"net/url"
	"strconv"
	"strings"
)

type PythonService struct {
	api *gin.Engine
}

func (svc *PythonService) Init() {
	svc.api.GET("/python", svc.getPythonList)
}

func (svc *PythonService) getPythonList(c *gin.Context) {
	// query
	query := c.Query("query")
	pagination := controllers.MustGetPagination(c)

	// validate
	if query == "" {
		controllers.HandleErrorBadRequest(c, errors.New("empty query"))
		return
	}

	// perform request
	requestUrl := fmt.Sprintf("https://pypi.org/search?page=%d&q=%s", pagination.Page, url.QueryEscape(query))
	ua := req.Header{"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.61 Safari/537.36"}
	res, err := req.Get(requestUrl, ua)
	if err != nil {
		_, _ = c.Writer.Write(res.Bytes())
		_ = c.AbortWithError(res.Response().StatusCode, err)
		return
	}

	// response bytes
	data, err := res.ToBytes()
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}
	buf := bytes.NewBuffer(data)

	// parse html
	doc, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// dependencies
	var deps []models.Dependency
	doc.Find(".left-layout__main > form ul > li").Each(func(i int, s *goquery.Selection) {
		d := models.Dependency{
			Name:        s.Find(".package-snippet__name").Text(),
			Version:     s.Find(".package-snippet__version").Text(),
			Description: s.Find(".package-snippet__description").Text(),
		}
		deps = append(deps, d)
	})

	// total
	totalStr := doc.Find(".left-layout__main .split-layout p > strong").Text()
	totalStr = strings.ReplaceAll(totalStr, ",", "")
	total, _ := strconv.Atoi(totalStr)

	controllers.HandleSuccessWithListData(c, deps, total)
}

func NewPythonService(api *gin.Engine) (svc *PythonService) {
	svc = &PythonService{
		api: api,
	}

	return svc
}
