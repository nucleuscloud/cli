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
        background-color: #FBFDF4;
    }

    .header {
        background-color: #FBFDF4;
    }

    .logo {
        height: 30%;
        width: 30%;
        border-radius: 10px;
        display: block;
        margin-left: auto;
        margin-right: auto
    }

    h1 {
        font-family: "Courier New", Courier, monospace;
        font-size: 36px;
        letter-spacing: -0.4px;
        word-spacing: 2px;
        color: #2F48FF;
        font-weight: normal;
        text-transform: capitalize;
        text-align: center;
    }

    p {
        font-family: "Courier New", Courier, monospace;
        font-size: 16px;
        letter-spacing: -0.4px;
        word-spacing: 2px;
        color: #489AFF;
        font-weight: normal;
        text-transform: capitalize;
        text-align: center;
    }

    .nucleusLogo {
        height: '40px';
        width: 40px;
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
<div><a href="https:nucleuscloud.com"><img class='nucleusLogo' src="https://assets.nucleuscloud.com/favicon_transparent.ico"></a></div>
    <div class='successText'>
        <h1>Login Success!</h1>
        <p>You've successfully logged in to Nucleus CLI.</p>
	  <p>You may now close this window and return to your terminal.</p>
    </div>
        <div>
  <img class='logo' src="https://assets.nucleuscloud.com/loginPicFinal.jpg">
    </div>
	`

	loginPageError = `
    <div><a href="https:nucleuscloud.com"><img class='nucleusLogo' src="https://assets.nucleuscloud.com/favicon_transparent.ico"></a></div>
    <div class='successText'>
        <h1>There was a problem logging you in!</h1>
        <p class="error-text">Error Code: {{ .ErrorCode }}</p>
        <p class="error-text">Error Description: {{ .ErrorDescription }}</p>
    </div>
    <div>
        <img class='logo' src="https://assets.nucleuscloud.com/angryDarth.jpg">
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

// returns a template with the header and footer templates added in
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
