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
		probeFailedDueToMissingSelector = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_failed_due_to_missing_selector",
			Help: "Indicates that selector doesn't became visible",
		})
	)
	registry.MustRegister(probeFailedDueToRegex)
	registry.MustRegister(probeFailedDueToMissingSelector)

	chromeConfig := module.CHROME

	targetURL, err := url.Parse(target)
	if err != nil {
		level.Error(logger).Log("msg", "Could not parse target URL", "err", err)
		return false
	}

	chromeCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	if err := chromedp.Run(chromeCtx, chromedp.Navigate(targetURL.String())); err != nil {
		level.Error(logger).Log("msg", "Could not run Chrome", "err", err)
		return false
	}

	if chromeConfig.TextSelector != "" {
		var res string
		if err = chromedp.Run(chromeCtx, chromedp.Text(chromeConfig.TextSelector, &res, chromedp.NodeVisible)); err != nil {
			level.Error(logger).Log("msg", "Failed to get text by selector", "selector", chromeConfig.TextSelector, "err", err)
			return false
		}
		res = strings.ToLower(res)
		level.Info(logger).Log("msg", "Found", "text", res, "selector", chromeConfig.TextSelector)

		if len(chromeConfig.FailIfTextMatchesRegexp) > 0 {
			success = matchRegex(res, chromeConfig, logger)
			if success {
				probeFailedDueToRegex.Set(0)
			} else {
				probeFailedDueToRegex.Set(1)
			}
		}
	}

	if chromeConfig.WaitVisibleSelector != "" {
		if err = chromedp.Run(chromeCtx, chromedp.WaitVisible(chromeConfig.WaitVisibleSelector)); err != nil {
			level.Error(logger).Log("msg", "Failed on waiting for selector to became visible", "selector", chromeConfig.WaitVisibleSelector, "err", err)
			probeFailedDueToMissingSelector.Set(1)
			return false
		}
		probeFailedDueToMissingSelector.Set(0)
		success = true
	}

	return
}
