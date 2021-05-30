package router

import (
	"strings"

	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/sys/tag"
)

type RuleGroup struct {
	RouteId          string `yaml:"routeId"`
	RouteName        string `yaml:"routeName"`
	RouteDesc        string `yaml:"routeDesc"`
	MicroserivceId   string `yaml:"microserivceId"`
	RuleList         []Rule `yaml:"ruleList"`
	NamespaceId      string `yaml:"namespaceId"`
	MicroserviceName string `yaml:"microserviceName"`
	FallbackStatus   bool   `yaml:"fallbackStatus"`
}

type Rule struct {
	RouteRuleId string    `yaml:"routeRuleId"`
	RouteId     string    `yaml:"routeId"`
	TagList     []TagRule `yaml:"tagList"`
	DestList    []Dest    `yaml:"destList"`
}

type Dest struct {
	DestId       string     `yaml:"destId"`
	DestWeight   int64      `yaml:"destWeight"`
	DestItemList []DestItem `yaml:"destItemList"`
	RouteRuleId  string     `yaml:"routeRuleId"`
}

type DestItem struct {
	RouteDestItemId string `yaml:"routeDestItemId"`
	RouteDestId     string `yaml:"routeDestId"`
	DestItemField   string `yaml:"destItemField"`
	DestItemValue   string `yaml:"destItemValue"`
}

type TagRule struct {
	TagID       string `yaml:"tagID"`
	TagType     string `yaml:"tagType"`
	TagField    string `yaml:"tagField"`
	TagOperator string `yaml:"tagOperator"`
	TagValue    string `yaml:"tagValue"`
	RouteRuleId string `yaml:"routeRuleId"`
}

func (rule Rule) toCommonTagRule() tag.Rule {
	tagRule := tag.Rule{
		Expression: tag.AND,
	}
	for _, routeTag := range rule.TagList {
		var t tag.Tag
		field := routeTag.TagField
		if routeTag.TagType != "U" {
			switch field {
			case "source.application.id":
				field = meta.ApplicationID
			case "source.group.id":
				field = meta.GroupID
			case "source.connection.ip":
				field = meta.ConnnectionIP
			case "source.application.version":
				field = meta.ApplicationVersion
			case "source.service.name":
				field = meta.ServiceName
			case "destination.interface":
				field = "destination.interface"
			case "request.http.method":
				field = "request.http.method"
			case "source.namespace.service.name":
				values := strings.SplitN(routeTag.TagValue, "/", 2)
				if len(values) != 2 {
					continue
				}
				t.Field = meta.Namespace
				t.Operator = routeTag.TagOperator
				t.Type = tag.TypeSys
				t.Value = values[0]
				tagRule.Tags = append(tagRule.Tags, t)

				t.Field = meta.ServiceName
				t.Operator = routeTag.TagOperator
				t.Type = tag.TypeSys
				t.Value = values[1]
				tagRule.Tags = append(tagRule.Tags, t)
				continue
			default:
			}
		}
		t.Field = field
		t.Operator = routeTag.TagOperator
		if routeTag.TagType == "U" {
			t.Type = tag.TypeUser
		} else {
			t.Type = tag.TypeSys
		}
		t.Value = routeTag.TagValue
		tagRule.Tags = append(tagRule.Tags, t)
	}
	return tagRule
}
