package mapper

import (
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func Healthcheck(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	if svc.HealthCheck == nil {
		return nil
	}
	hc := svc.HealthCheck
	if hc.Disable {
		return nil
	}
	var dirs []c2qtypes.Directive

	if len(hc.Test) > 0 {
		if strings.EqualFold(hc.Test[0], "NONE") {
			return nil
		}
		if strings.EqualFold(hc.Test[0], "CMD-SHELL") {
			cmd := strings.Join(hc.Test[1:], " ")
			dirs = append(dirs, c2qtypes.Directive{Key: "HealthCmd", Values: []string{"/bin/sh -c " + cmd}})
		} else if strings.EqualFold(hc.Test[0], "CMD") {
			dirs = append(dirs, c2qtypes.Directive{Key: "HealthCmd", Values: []string{strings.Join(hc.Test[1:], " ")}})
		}
	}

	if hc.Interval != nil {
		dirs = append(dirs, c2qtypes.Directive{Key: "HealthInterval", Values: []string{time.Duration(*hc.Interval).String()}})
	}
	if hc.Timeout != nil {
		dirs = append(dirs, c2qtypes.Directive{Key: "HealthTimeout", Values: []string{time.Duration(*hc.Timeout).String()}})
	}
	if hc.Retries != nil {
		dirs = append(dirs, c2qtypes.Directive{Key: "HealthRetries", Values: []string{strconv.FormatUint(*hc.Retries, 10)}})
	}
	if hc.StartPeriod != nil {
		dirs = append(dirs, c2qtypes.Directive{Key: "HealthStartPeriod", Values: []string{time.Duration(*hc.StartPeriod).String()}})
	}
	if hc.StartInterval != nil {
		dirs = append(dirs, c2qtypes.Directive{Key: "HealthStartupInterval", Values: []string{time.Duration(*hc.StartInterval).String()}})
	}

	return dirs
}
