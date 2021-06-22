package lane

import (
	"time"

	"github.com/tencentyun/tsf-go/pkg/sys/tag"
)

type LaneRule struct {
	ID           string    `yaml:"ruleId"` //本身LaneRule的ID
	Name         string    `yaml:"ruleName"`
	Enable       bool      `yaml:"enable"`
	LaneID       string    `yaml:"laneId"` //对于的泳道信息的ID，不是LaneRuleID
	Priority     int64     `yaml:"priority"`
	TagList      []TagRule `yaml:"ruleTagList"`
	Relationship string    `yaml:"ruleTagRelationship"`
	CreateTime   time.Time `yaml:"createTime"`
}

type TagRule struct {
	ID       string `yaml:"tagId"` //本身tag的ID
	Name     string `yaml:"tagName"`
	Operator string `yaml:"tagOperator"`
	Value    string `yaml:"tagValue"`
}

func (rule LaneRule) toCommonTagRule() tag.Rule {
	var tagRule tag.Rule
	tagRule.ID = rule.ID
	if rule.Relationship == "RELEATION_OR" {
		tagRule.Expression = tag.OR
	} else {
		tagRule.Expression = tag.AND
	}
	for _, routeTag := range rule.TagList {
		var t tag.Tag
		t.Field = routeTag.Name
		t.Operator = routeTag.Operator
		t.Type = tag.TypeUser
		t.Value = routeTag.Value
		tagRule.Tags = append(tagRule.Tags, t)
	}
	return tagRule
}

type LaneInfo struct {
	ID         string      `yaml:"laneId"`
	Name       string      `yaml:"laneName"`
	GroupList  []LaneGroup `yaml:"laneGroupList"`
	CreateTime time.Time   `yaml:"createTime"`
}

type LaneGroup struct {
	ApplicationID   string `yaml:"applicationId"`
	ApplicationName string `yaml:"applicationName"`
	ClusterType     string `yaml:"clusterType"`
	Entrance        bool   `yaml:"entrance"`
	GroupID         string `yaml:"groupId"`
	NamespaceID     string `yaml:"namespaceId"`
	GroupName       string `yaml:"groupName"`
}
