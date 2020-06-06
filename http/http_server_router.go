/**
* @program: lemo
*
* @description:
*
* @author: lemo
*
* @create: 2019-11-25 11:29
**/

package http

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Lemo-yxk/lemo"
	"github.com/Lemo-yxk/lemo/caller"
	"github.com/Lemo-yxk/lemo/container/tire"
	"github.com/Lemo-yxk/lemo/exception"
)

type groupFunction func(handler *RouteHandler)

type function func(stream *Stream) exception.Error

type before func(stream *Stream) (lemo.Context, exception.Error)

type after func(stream *Stream) exception.Error

var globalBefore []before
var globalAfter []after

func SetBefore(before ...before) {
	globalBefore = append(globalBefore, before...)
}

func SetAfter(after ...after) {
	globalAfter = append(globalAfter, after...)
}

type group struct {
	path   string
	before []before
	after  []after
	router *Router
}

func (group *group) Route(path string) *group {
	group.path = path
	return group
}

func (group *group) Before(before ...before) *group {
	group.before = append(group.before, before...)
	return group
}

func (group *group) After(after ...after) *group {
	group.after = append(group.after, after...)
	return group
}

func (group *group) Handler(fn groupFunction) {

	if group.path == "" {
		panic("group path can not empty")
	}

	fn(&RouteHandler{group: group})
}

type RouteHandler struct {
	group *group
}

func (handler *RouteHandler) Route(method string, path string) *route {
	return &route{path: path, method: method, group: handler.group}
}

func (handler *RouteHandler) Get(path string) *route {
	return handler.Route("GET", path)
}

func (handler *RouteHandler) Post(path string) *route {
	return handler.Route("POST", path)
}

func (handler *RouteHandler) Delete(path string) *route {
	return handler.Route("DELETE", path)
}

func (handler *RouteHandler) Put(path string) *route {
	return handler.Route("PUT", path)
}

func (handler *RouteHandler) Patch(path string) *route {
	return handler.Route("PATCH", path)
}

func (handler *RouteHandler) Option(path string) *route {
	return handler.Route("OPTION", path)
}

type route struct {
	path        string
	method      string
	before      []before
	after       []after
	passBefore  bool
	forceBefore bool
	passAfter   bool
	forceAfter  bool
	group       *group
}

func (route *route) Before(before ...before) *route {
	route.before = append(route.before, before...)
	return route
}

func (route *route) PassBefore() *route {
	route.passBefore = true
	return route
}

func (route *route) ForceBefore() *route {
	route.forceBefore = true
	return route
}

func (route *route) After(after ...after) *route {
	route.after = append(route.after, after...)
	return route
}

func (route *route) PassAfter() *route {
	route.passAfter = true
	return route
}

func (route *route) ForceAfter() *route {
	route.forceAfter = true
	return route
}

func (route *route) Handler(fn function) {

	if route.path == "" || route.method == "" {
		panic("route path or method can not empty")
	}

	file, line := caller.Caller(1)

	var g = route.group

	var router = route.group.router

	var method = strings.ToUpper(route.method)

	if g == nil {
		g = new(group)
	}

	var path = router.formatPath(g.path + route.path)

	if router.tire == nil {
		router.tire = new(tire.Tire)
	}

	var hba = &node{}

	hba.info = file + ":" + strconv.Itoa(line)

	hba.function = fn

	hba.before = append(g.before, route.before...)
	if route.passBefore {
		hba.before = nil
	}
	if route.forceBefore {
		hba.before = route.before
	}

	hba.after = append(g.after, route.after...)
	if route.passAfter {
		hba.after = nil
	}
	if route.forceAfter {
		hba.after = route.after
	}

	hba.before = append(hba.before, globalBefore...)
	hba.after = append(hba.after, globalAfter...)

	hba.method = method

	hba.route = []byte(path)

	router.tire.Insert(path, hba)
}

type Router struct {
	IgnoreCase   bool
	tire         *tire.Tire
	prefixPath   string
	staticPath   string
	defaultIndex string
}

func (router *Router) GetAllRouters() []*node {
	var res []*node
	var tires = router.tire.GetAllValue()
	for i := 0; i < len(tires); i++ {
		res = append(res, tires[i].Data.(*node))
	}
	return res
}

func (router *Router) SetDefaultIndex(index string) {
	router.defaultIndex = index
}

func (router *Router) SetStaticPath(prefixPath string, staticPath string) {

	if prefixPath == "" {
		panic("prefixPath can not be empty")
	}

	if staticPath == "" {
		panic("staticPath can not be empty")
	}

	absStaticPath, err := filepath.Abs(staticPath)
	if err != nil {
		panic(err)
	}

	info, err := os.Stat(absStaticPath)
	if err != nil {
		panic(err)
	}

	if !info.IsDir() {
		panic("staticPath is not a dir")
	}

	router.prefixPath = prefixPath
	router.staticPath = absStaticPath
	router.defaultIndex = "index.html"
}

func (router *Router) Group(path string) *group {

	var group = new(group)

	group.Route(path)

	group.router = router

	return group
}

func (router *Router) Route(method string, path string) *route {
	return (&RouteHandler{group: router.Group("")}).Route(method, path)
}

func (router *Router) getRoute(method string, path string) (*tire.Tire, []byte) {

	if router.tire == nil {
		return nil, nil
	}

	method = strings.ToUpper(method)

	path = router.formatPath(path)

	var pathB = []byte(path)

	var t = router.tire.GetValue(pathB)

	if t == nil {
		return nil, nil
	}

	if t.Data.(*node).method != method {
		return nil, nil
	}

	return t, pathB
}

func (router *Router) formatPath(path string) string {
	if router.IgnoreCase {
		path = strings.ToLower(path)
	}
	return path
}

type node struct {
	info     string
	route    []byte
	method   string
	function function
	before   []before
	after    []after
}