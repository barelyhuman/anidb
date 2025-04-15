package main

import "strings"

type QueryStringBuilder struct {
	b      strings.Builder
	params []any
}

func (q *QueryStringBuilder) Query(s string) *QueryStringBuilder {
	q.b.WriteString(s)
	return q
}

func (q *QueryStringBuilder) Param(val any) *QueryStringBuilder {
	q.b.WriteString("?")
	q.params = append(q.params, val)
	return q
}

func (q *QueryStringBuilder) Get() (string, []any) {
	return q.b.String(), q.params
}
