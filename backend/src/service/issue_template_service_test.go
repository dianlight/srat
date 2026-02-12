package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

type IssueTemplateServiceTestSuite struct {
	suite.Suite
	service *IssueTemplateService
}

func (suite *IssueTemplateServiceTestSuite) SetupTest() {
	suite.service = &IssueTemplateService{
		httpClient: &http.Client{},
	}
	httpmock.ActivateNonDefault(suite.service.httpClient)
}

func (suite *IssueTemplateServiceTestSuite) TearDownTest() {
	httpmock.DeactivateAndReset()
}

func (suite *IssueTemplateServiceTestSuite) TestFetchIssueTemplate_Success() {
	yamlContent := `
name: Bug Report
description: Report a bug
title: "[BUG] "
labels: ["bug"]
body:
  - type: markdown
    attributes:
      value: "Thanks for reporting!"
`
	httpmock.RegisterResponder("GET", issueTemplateURL,
		httpmock.NewStringResponder(200, yamlContent))

	template, err := suite.service.fetchIssueTemplate(context.Background())
	suite.Require().NoError(err)
	suite.NotNil(template)
	suite.Equal("Bug Report", template.Name)
}

func (suite *IssueTemplateServiceTestSuite) TestFetchIssueTemplate_HTTPError() {
	httpmock.RegisterResponder("GET", issueTemplateURL,
		httpmock.NewStringResponder(404, "Not Found"))

	template, err := suite.service.fetchIssueTemplate(context.Background())
	suite.Require().Error(err)
	suite.Nil(template)
	suite.Contains(err.Error(), "unexpected status code: 404")
}

func (suite *IssueTemplateServiceTestSuite) TestGetTemplate() {
	suite.service.template = &dto.IssueTemplate{Name: "Test"}
	template, err := suite.service.GetTemplate()
	suite.Require().NoError(err)
	suite.Equal("Test", template.Name)

	suite.service.template = nil
	template, err = suite.service.GetTemplate()
	suite.Require().Error(err)
	suite.Nil(template)
}

func TestIssueTemplateServiceTestSuite(t *testing.T) {
	suite.Run(t, new(IssueTemplateServiceTestSuite))
}
