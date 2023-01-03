package utils

import (
	"fmt"
	"html/template"
	"io"
)

const (
	header = `
<!DOCTYPE html>
<head>
  <title>{{ .Title }}</title>
	<link rel="icon" type="image/png" href="https://assets.nucleuscloud.com/favicon_transparent.ico" />
  <style>
    body {
      background-color: #BBC;
			margin: 8px;
			display: block;
    }
    #content {
    }
    #footer {
      font-size: 0.8em;
    }
		.error-text {
			font-weight: bold;
		}
  </style>
</head>
<body>
  <div id="content">
	`

	footer = `
	</div>
  <div id="footer"></div>
</body>
</html>
	`

	loginPageSuccess = `
<div>
 <h1>Login Success!</h1>
 <div>
	<p>You've successfully logged in to Nucleus CLI.</p>
	<p>You may now close this window and return to your terminal.</p>
 </div>
</div>
	`

	loginPageError = `
<div>
<h1>There was a problem logging you in!</h1>
<p class="error-text">Error Code: {{ .ErrorCode }}</p>
<p class="error-text">Error Description: {{ .ErrorDescription }}</p>
</div>
	`
)

// wraps page with header and footer
func wrapPage(contents string) string {
	return fmt.Sprintf(
		`
{{ template "header" . }}
%s
{{ template "footer" . }}
`, contents,
	)
}

type LoginPageData struct {
	Title string
}

func RenderLoginSuccessPage(wr io.Writer, data LoginPageData) error {
	pageTmpl, err := getHtmlPage()
	if err != nil {
		return err
	}
	pageTmpl, err = pageTmpl.New("login").Parse(wrapPage(loginPageSuccess))
	if err != nil {
		return err
	}
	return pageTmpl.ExecuteTemplate(wr, "login", data)
}

type LoginPageErrorData struct {
	Title string

	ErrorCode        string
	ErrorDescription string
}

func RenderLoginErrorPage(wr io.Writer, data LoginPageErrorData) error {
	pageTmpl, err := getHtmlPage()
	if err != nil {
		return err
	}
	pageTmpl, err = pageTmpl.New("login").Parse(wrapPage(loginPageError))
	if err != nil {
		return err
	}
	return pageTmpl.ExecuteTemplate(wr, "login", data)
}

func getHtmlPage() (*template.Template, error) {
	templ, err := template.New("header").Parse(header)
	if err != nil {
		return nil, err
	}
	templ, err = templ.New("footer").Parse(footer)
	if err != nil {
		return nil, err
	}
	return templ, nil
}
