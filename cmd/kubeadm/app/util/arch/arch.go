/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020-04-20  @author lizhen
 */

package arch

import (
	"fmt"
	"runtime"
	"strings"
)

func archSuffix() string {
	if strings.EqualFold(runtime.GOARCH, "amd64") {
		return ""
	}
	return fmt.Sprintf("-%s", runtime.GOARCH)
}

var (
	ImageSuffix = archSuffix()
)
