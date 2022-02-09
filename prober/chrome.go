// Copyright 2016 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prober

import (
	"context"
	"github.com/chromedp/chromedp"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"net/url"
	"strings"
)

func matchRegex(body string, chromeConfig config.CHROMEProbe, logger log.Logger) bool {
	for _, expression := range chromeConfig.FailIfTextMatchesRegexp {
		if expression.Regexp.MatchString(body) {
			level.Error(logger).Log("msg", "Text matched regular expression", "regexp", expression)
			return false
		}
	}
	return true
}

func CHROMEProbe(ctx context.Context, target string, module config.Module, registry *prometheus.Registry, logger log.Logger) (success bool) {
	var (
		probeFailedDueToRegex = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_failed_due_to_regex",
			Help: "Indicates if probe failed due to regex",
		})
	)
	registry.MustRegister(probeFailedDueToRegex)

	chromeConfig := module.CHROME

	targetURL, err := url.Parse(target)
	if err != nil {
		level.Error(logger).Log("msg", "Could not parse target URL", "err", err)
		return false
	}

	chromeCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()
	var res string
	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(targetURL.String()),
		chromedp.Text(chromeConfig.TextSelector, &res, chromedp.NodeVisible),
	)
	res = strings.ToLower(res)
	if err != nil {
		level.Error(logger).Log("msg", "Could not run Chrome", "err", err)
		return false
	}

	level.Info(logger).Log("msg", "Found", "text", res, "selector", chromeConfig.TextSelector)

	if len(chromeConfig.FailIfTextMatchesRegexp) > 0 {
		success = matchRegex(res, chromeConfig, logger)
		if success {
			probeFailedDueToRegex.Set(0)
		} else {
			probeFailedDueToRegex.Set(1)
		}
	}

	return
}
