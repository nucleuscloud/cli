package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cli/oauth/api"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Logs a user into their Nucleus account.",
	Long:  `Logs a user into their Nucleus account. `,

	RunE: func(cmd *cobra.Command, args []string) error {

		template := &promptui.SelectTemplates{
			Label: "{{ . }}:",
			Help:  " ",
		}

		loginSelect := promptui.Select{
			Label:     "Use the arrow keys to select the provider you want to login with below",
			Templates: template,
			Items:     []string{"Github"}, //add to the object to add in new providers
		}

		result, _, err := loginSelect.Run()

		if err != nil {
			fmt.Printf("The select prompt failed")
		}

		c := httpClient()

		if result == 0 {
			//this is the first index in the Items object from the loginSelect variable

			scope := []string{"repo:status", "user", "read:org"} //define the scopes that we want to access, link to scopes: https://docs.github.com/en/developers/apps/building-oauth-apps/scopes-for-oauth-apps, user will return all public user information

			gitAuth, err := githubAuth(c, scope) //kicks off the github oauth procss
			if err != nil {
				fmt.Printf("Error in calling the github authentication endpoint")
			}

			fmt.Println("First, copy your one-time code: ", gitAuth.UserCode)

			cliPrompt("\nThen press [Enter] to continue in the web browser...", "")

			webbrowser.Open(gitAuth.VerificationURI)

			access_token, err := PollToken(c, gitAuth) //returns a valid oauth token
			if err != nil {
				fmt.Printf("Error getting the oauth access token")
			}

			//use oauth token to be able to call github APIs and get info about the user
			orgURL := "https://api.github.com/user"

			resp, err := getUserInfo(c, access_token, orgURL)
			if err != nil {
				fmt.Printf("there is an error with the struct encoding")
			}

			fmt.Println("final user object", resp.Email, resp.Id, resp.Login, resp.Profile)

		}

		return nil

	},
}

type GithubResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`

	timeNow   func() time.Time
	timeSleep func(time.Duration)
}

type UserInfo struct {
	Login   string `json:"login"`
	Id      int    `json:"id"`
	Profile string `json:"html_url"`
	Email   UserEmail
	Org     UserOrg
}

type UserEmail []struct {
	Email string `json:"email"`
}

type UserOrg []struct {
	Org string `json:"login"`
}

func httpClient() *http.Client {
	client := &http.Client{Timeout: 10 * time.Second}
	return client
}

func githubAuth(c *http.Client, scope []string) (*GithubResponse, error) {

	clientID := "4fecd90b045067eac7e6"
	baseUrl := "https://github.com/login/device/code"

	fmt.Println(strings.Join(scope, ","))

	resp, err := api.PostForm(c, baseUrl, url.Values{
		"client_id": {clientID},
		"scope":     {strings.Join(scope, " ")},
	})
	if err != nil {
		return nil, err
	}

	interval, err := strconv.Atoi(resp.Get("interval"))
	if err != nil {
		fmt.Printf("There was an error in parsing the string")
	}

	expires_in, err := strconv.Atoi(resp.Get("expires_in"))
	if err != nil {
		fmt.Printf("There was an error in parsing the string")
	}

	return &GithubResponse{
		DeviceCode:      resp.Get("device_code"),
		UserCode:        resp.Get("user_code"),
		VerificationURI: resp.Get("verification_uri"),
		Interval:        interval,
		ExpiresIn:       expires_in,
	}, err
}

func PollToken(c *http.Client, code *GithubResponse) (*api.AccessToken, error) {

	pollUrl := "https://github.com/login/oauth/access_token"

	clientID := "4fecd90b045067eac7e6"

	const grantType = "urn:ietf:params:oauth:grant-type:device_code"

	timeNow := code.timeNow
	if timeNow == nil {
		timeNow = time.Now
	}
	timeSleep := code.timeSleep
	if timeSleep == nil {
		timeSleep = time.Sleep
	}

	checkInterval := time.Duration(code.Interval) * time.Second
	expiresAt := timeNow().Add(time.Duration(code.ExpiresIn) * time.Second)
	ErrTimeout := errors.New("authentication timed out")

	for {
		timeSleep(checkInterval)

		resp, err := api.PostForm(c, pollUrl, url.Values{
			"client_id":   {clientID},
			"device_code": {code.DeviceCode},
			"grant_type":  {grantType},
		})
		if err != nil {
			return nil, err
		}

		var apiError *api.Error
		token, err := resp.AccessToken()

		if err == nil {
			fmt.Println("this is the access token: ", token)
			return token, nil
		} else if !(errors.As(err, &apiError) && apiError.Code == "authorization_pending") {
			return nil, err
		}

		if timeNow().After(expiresAt) {
			return nil, ErrTimeout
		}
	}
}

func getUserEmail(c *http.Client, token *api.AccessToken) (*UserEmail, error) {
	//have to call the email API directly since some users don't make their emails public, this will always return an email even if set to private
	url := "https://api.github.com/user/emails"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+token.Token)

	resp, err := c.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(body))

	var userEmail UserEmail

	json.Unmarshal(body, &userEmail)

	fmt.Println("this is the user email", userEmail)

	return &userEmail, err

}

func getUserOrg(c *http.Client, token *api.AccessToken) (*UserOrg, error) {
	//attempt to get the organizations that the user belongs to
	url := "https://api.github.com/user/orgs"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+token.Token)

	resp, err := c.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(body))

	var userOrg UserOrg

	json.Unmarshal(body, &userOrg)

	fmt.Println("this is the user orgs", userOrg)

	return &userOrg, err
}

func getUserInfo(c *http.Client, token *api.AccessToken, url string) (*UserInfo, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+token.Token)

	resp, err := c.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(body))

	email, err := getUserEmail(c, token)
	if err != nil {
		fmt.Println(err)
	}

	org, err := getUserOrg(c, token)
	if err != nil {
		fmt.Println(err)
	}

	var userInfo UserInfo

	json.Unmarshal(body, &userInfo)

	return &UserInfo{
		Login:   userInfo.Login,
		Profile: userInfo.Profile,
		Id:      userInfo.Id,
		Email:   *email,
		Org:     *org,
	}, err
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
