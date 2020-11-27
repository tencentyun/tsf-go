package authenticator

import (
	"strings"

	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/sys/tag"
)

type AuthConfig struct {
	Rules []AuthRule `yaml:"rules"`
	Type  string     `yaml:"type"`
}

type AuthRule struct {
	ID      string `yaml:"ruleId"`
	Name    string `yaml:"ruleName"`
	Tags    []Tag  `yaml:"tags"`
	tagRule tag.Rule
}

type Tag struct {
	ID       string `yaml:"tagId"`
	Type     string `yaml:"tagType"`
	Field    string `yaml:"tagField"`
	Operator string `yaml:"tagOperator"`
	Value    string `yaml:"tagValue"`
}

func (rule *AuthRule) genTagRules() {
	var tagRule tag.Rule
	tagRule.Expression = tag.AND
	tagRule.ID = rule.ID
	for _, authTag := range rule.Tags {
		var t tag.Tag
		if authTag.Type == "S" && authTag.Field == "source.namespace.service.name" {
			values := strings.SplitN(authTag.Value, "/", 2)
			if len(values) != 2 {
				continue
			}
			t.Field = meta.Namespace
			t.Operator = authTag.Operator
			t.Type = tag.TypeSys
			t.Value = values[0]
			tagRule.Tags = append(tagRule.Tags, t)

			t.Field = meta.ServiceName
			t.Operator = authTag.Operator
			t.Type = tag.TypeSys
			t.Value = values[1]
			tagRule.Tags = append(tagRule.Tags, t)
			continue
		}
		t.Field = authTag.Field
		if strings.HasPrefix(t.Field, meta.PrefixDest) {
			t.Field = strings.TrimPrefix(t.Field, meta.PrefixDest)
		}
		t.Operator = authTag.Operator
		if authTag.Type == "S" {
			t.Type = tag.TypeSys
		} else {
			t.Type = tag.TypeUser
		}
		t.Value = authTag.Value
		tagRule.Tags = append(tagRule.Tags, t)
	}
	rule.tagRule = tagRule
}
