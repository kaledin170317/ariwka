package grpcui

import (
	"context"
	"net/http"
	"strings"

	"github.com/fullstorydev/grpcui/standalone"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Serve(address, target string) error {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	handler, err := standalone.HandlerViaReflection(context.Background(), conn, target)
	if err != nil {
		return err
	}

	return http.ListenAndServe(address, matchingCSRFCookie(handler))
}

func matchingCSRFCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/invoke/") {
			token := r.Header.Get("x-grpcui-csrf-token")
			cookies := r.Cookies()

			var selected *http.Cookie

			for _, cookie := range cookies {
				if cookie.Name == "_grpcui_csrf_token" && cookie.Value == token && token != "" {
					selected = cookie
				}
			}

			if selected != nil {
				values := []string{selected.String()}

				for _, cookie := range cookies {
					if cookie.Name != "_grpcui_csrf_token" {
						values = append(values, cookie.String())
					}
				}

				r.Header.Set("Cookie", strings.Join(values, "; "))
			}
		}

		next.ServeHTTP(w, r)
	})
}
