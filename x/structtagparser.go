// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Copy From: go.mongodb.org/mongo-driver/v2@v2.8.0/bson/struct_tag_parser.go

// Package x
/*
Derived from: go.mongodb.org/mongo-driver/v2@v2.8.0/bson/struct_tag_parser.go

Modifications:
  - Line 73&85: Changed `key := strings.ToLower(sf.Name)` to `key := sf.Name`
    That's it. One line. One character of intent. Everything else is collateral.

Before: UserName → username (because someone decided lowercase was a moral imperative)
After:  UserName → UserName (because that's what you typed)

The rest of this file is essentially a faithful copy of the upstream codec,
except now it treats your identifiers with the basic dignity they deserve.
We didn't rewrite the wheel. We just stopped it from deflating itself on purpose.

Special thanks to the v2 team for making "don't touch my field names" a
feature that requires forking half the codec package. Very motivational. 🙃
*/
//
/*
基于: go.mongodb.org/mongo-driver/v2@v2.8.0/bson/struct_tag_parser.go

修改内容:
  - 第 73&85 行：将 `key := strings.ToLower(sf.Name)` 改为 `key := sf.Name`
    就这一行。一个字符的意图。其余全是连带的。

之前: UserName → username（因为有人认定小写是一种道德义务）
之后: UserName → UserName（因为那就是你敲出来的样子）

这个文件的其他部分基本是对上游 codec 的忠实复刻，只不过现在它给了你的标识符应有的基本尊重。
我们没有重写轮子，只是阻止了它故意漏气。

特别感谢 v2 团队让"别碰我的字段名"这样一个诉求变成了需要 fork 半个 codec 包才能实现的功能。非常有创作动力。🙃
*/
package x

import (
	"reflect"
	"strings"
)

// StructTags represents the struct tag fields that the StructCodec uses during
// the encoding and decoding process.
//
// In the case of a struct, the lowercased field name is used as the key for each exported
// field but this behavior may be changed using a struct tag. The tag may also contain flags to
// adjust the marshalling behavior for the field.
//
// The properties are defined below:
//
//	OmitEmpty  Only include the field if it's not set to the zero value for the type or to
//	           empty slices or maps.
//
//	MinSize    Marshal an integer of a type larger than 32 bits value as an int32, if that's
//	           feasible while preserving the numeric value.
//
//	Truncate   When unmarshaling a BSON double, it is permitted to lose precision to fit within
//	           a float32.
//
//	Inline     Inline the field, which must be a struct or a map, causing all of its fields
//	           or keys to be processed as if they were part of the outer struct. For maps,
//	           keys must not conflict with the bson keys of other struct fields.
//
//	Skip       This struct field should be skipped. This is usually denoted by parsing a "-"
//	           for the name.
type StructTags struct {
	Name      string
	OmitEmpty bool
	MinSize   bool // todo not support
	Truncate  bool // todo not support
	Inline    bool
	Skip      bool
}

// ParseStructTags is the StructTagParser used by the StructCodec by default.
// It will handle the bson struct tag. See the documentation for StructTags to see
// what each of the returned fields means.
//
// If there is no name in the struct tag fields, the struct field name is lowercased.
// The tag formats accepted are:
//
//	"[<key>][,<flag1>[,<flag2>]]"
//
//	`(...) bson:"[<key>][,<flag1>[,<flag2>]]" (...)`
//
// An example:
//
//	type T struct {
//	    A bool
//	    B int    "myb"
//	    C string "myc,omitempty"
//	    D string `bson:",omitempty" json:"jsonkey"`
//	    E int64  ",minsize"
//	    F int64  "myf,omitempty,minsize"
//	}
//
// A struct tag either consisting entirely of '-' or with a bson key with a
// value consisting entirely of '-' will return a StructTags with Skip true and
// the remaining fields will be their default values.
func ParseStructTags(sf reflect.StructField) (*StructTags, error) {
	return ParseStructTagsToLower(sf, false)
}

func ParseStructTagsToLower(sf reflect.StructField, toLower bool) (*StructTags, error) {
	key := sf.Name
	if toLower {
		key = strings.ToLower(sf.Name)
	}
	tag, ok := sf.Tag.Lookup("bson")
	if !ok && !strings.Contains(string(sf.Tag), ":") && len(sf.Tag) > 0 {
		tag = string(sf.Tag)
	}
	return parseTags(key, tag)
}

// ParseJSONStructTags has the same behavior as ParseStructTags
// but will also fallback to parsing the json tag instead on a field where the
// bson tag isn't available.
func ParseJSONStructTags(sf reflect.StructField) (*StructTags, error) {
	return ParseJSONStructTagsToLower(sf, false)
}

func ParseJSONStructTagsToLower(sf reflect.StructField, toLower bool) (*StructTags, error) {
	key := sf.Name
	if toLower {
		key = strings.ToLower(sf.Name)
	}
	tag, ok := sf.Tag.Lookup("bson")
	if !ok {
		tag, ok = sf.Tag.Lookup("json")
	}
	if !ok && !strings.Contains(string(sf.Tag), ":") && len(sf.Tag) > 0 {
		tag = string(sf.Tag)
	}

	return parseTags(key, tag)
}

func parseTags(key string, tag string) (*StructTags, error) {
	var st StructTags
	if tag == "-" {
		st.Skip = true
		return &st, nil
	}

	for idx, str := range strings.Split(tag, ",") {
		if idx == 0 && str != "" {
			key = str
		}
		switch str {
		case "omitempty":
			st.OmitEmpty = true
		case "minsize":
			st.MinSize = true
		case "truncate":
			st.Truncate = true
		case "inline":
			st.Inline = true
		}
	}

	st.Name = key

	return &st, nil
}

func ParseStruct(sf reflect.StructField, toLower bool, useJson bool) (*StructTags, error) {
	if useJson {
		return ParseJSONStructTagsToLower(sf, toLower)
	}

	return ParseStructTagsToLower(sf, toLower)
}
