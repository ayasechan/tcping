package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/imkira/go-interpol"
	"github.com/urfave/cli/v2"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}

}

func run() error {
	app := &cli.App{
		Name:      "tcping",
		UsageText: "tcping <domain> [port]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowAppHelp(c)
				return nil
			}
			domain := c.Args().Get(0)
			ip := resolve(domain)
			port := c.Args().Get(1)
			if port == "" {
				port = "443"
			}
			var seq, count int
			var max, min, sum, avg time.Duration
			var text string
			ticker := time.NewTicker(time.Second)
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt)
		loop:
			for {
				select {
				case <-ticker.C:
					select {
					case <-sig:
						break loop
					default:
					}
					seq += 1
					dur, err := tcping(fmt.Sprintf("%s:%s", domain, port))
					if err != nil {
						text = fstring(`from {domain}({ip}): {err}`, map[string]interface{}{
							"domain": domain,
							"ip":     ip,
							"err":    err,
						})
						fmt.Println(text)
						continue
					}
					count += 1
					sum += dur
					if min == 0 {
						min = dur
					}
					if dur > max {
						max = dur
					}
					if dur < min {
						min = dur
					}

					text = fstring(`from {domain}({ip}): seq={seq} time={ms}`, map[string]interface{}{
						"domain": domain,
						"ip":     ip,
						"seq":    seq,
						"ms":     dur,
					})
					fmt.Println(text)
				case <-sig:
					break loop
				}
			}

			fmt.Println("")
			text = `--- {domain} tcping statistics ---`
			text = fstring(text, map[string]interface{}{"domain": domain})
			fmt.Println(text)
			loss := (seq - count) * 100 / seq
			text = `{seq} packets transmitted, {count} received, {loss}% packet loss`
			text = fstring(text, map[string]interface{}{
				"seq":   seq,
				"count": count,
				"loss":  loss,
			})
			fmt.Println(text)
			text = `rtt min/avg/max = {min}/{avg}/{max}`
			if count != 0 {
				avg = sum / time.Duration(count)
			}
			text = fstring(text, map[string]interface{}{
				"min": min,
				"avg": avg,
				"max": max,
			})
			fmt.Println(text)
			return nil
		},
	}

	return app.Run(os.Args)
}

func tcping(addr string) (time.Duration, error) {
	return timeIt(func() error {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return err
		}
		defer conn.Close()
		return nil
	})

}

func timeIt(fn func() error) (time.Duration, error) {
	startAt := time.Now()
	err := fn()
	endAt := time.Now()

	return endAt.Sub(startAt), err

}

func fstring(text string, m map[string]interface{}) string {
	_m := make(map[string]string, len(m))
	for k, v := range m {
		switch v := v.(type) {
		case error:
			_m[k] = v.Error()
		case string:
			_m[k] = v
		case int, int64:
			_m[k] = fmt.Sprintf("%d", v)
		case float32, float64:
			_m[k] = fmt.Sprintf("%f", v)
		case time.Duration:
			_m[k] = fmt.Sprintf("%d ms", v.Milliseconds())
		case fmt.Stringer:
			_m[k] = v.String()
		default:
			_m[k] = fmt.Sprintf("%v", v)
		}
	}
	text, _ = interpol.WithMap(text, _m)

	return text
}

func resolve(domain string) net.IP {
	ip := net.ParseIP(domain)
	if ip != nil {
		return ip
	}
	ips, err := net.LookupIP(domain)
	if err != nil {
		panic(err)
	}
	return ips[0]
}
