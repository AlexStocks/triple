package tools

import (
	"github.com/dubbogo/triple/internal/codes"
	"github.com/dubbogo/triple/internal/status"
	"github.com/dubbogo/triple/pkg/config"
	"strings"
)

// AddDefaultOption fill default options to @opt
func AddDefaultOption(opt *config.Option) *config.Option {
	if opt == nil {
		opt = &config.Option{}
	}
	opt.SetEmptyFieldDefaultConfig()
	return opt
}

func GetServiceKeyAndUpperCaseMethodNameFromPath(path string) (string, string, error) {
	paramList := strings.Split(path, "/")
	if len(paramList) < 3 {
		return "", "", status.Errorf(codes.Internal, "invalid triple header path = %s", path)
	}
	methodName := paramList[2]
	if methodName == "" {
		return "", "", status.Errorf(codes.Internal, "invalid method name = %s", methodName)
	}
	methodName = strings.ToUpper(string(methodName[0])) + methodName[1:]
	return paramList[1], methodName, nil

}
