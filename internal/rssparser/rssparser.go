package rssparser

import (
	"html"
	"reflect"
)

type RSSFeed struct {
	Channel struct {
		Title	string	`xml:"title"`
		Link	string 	`xml:"link"`
		Description	string	`xml:"description"`
		Item	[]RSSItem	`xml:"item"`
	}	`xml:"channel"`
}

type RSSItem struct {
	Title	string	`xml:"title"`
	Link	string	`xml:"link"`
	Description	string `xml:"description"`
	PubDate	string	`xml:"pubDate"`
}

func Unescaper(v interface{}) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}
		
		switch field.Kind() {
		case reflect.String:
			field.SetString(html.UnescapeString(field.String()))

		case reflect.Struct:
			Unescaper(field.Addr().Interface())

		case reflect.Ptr:
			if !field.IsNil() && field.Elem().Kind() == reflect.Struct {
				Unescaper(field.Interface())
			}

		case reflect.Slice:
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				switch elem.Kind() {
				case reflect.String:
					elem.SetString(html.UnescapeString(elem.String()))
				case reflect.Struct:
					Unescaper(elem.Addr().Interface())
				case reflect.Ptr:
					if !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
						Unescaper(elem.Interface())
					}
				}
			}
		}
	}
}
