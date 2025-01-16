// Copyright 2025 Abhisheke Acharya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"time"

	"github.com/go-rod/rod/lib/proto"
)

// Lead represents a lead from apollo.io.
type Lead struct {
	Name      string `json:"name"      csv:"name"`
	Title     string `json:"title"     csv:"title"`
	Company   string `json:"company"   csv:"company"`
	Location  string `json:"location"  csv:"location"`
	Employees string `json:"employees" csv:"employees"`
	Industry  string `json:"industry"  csv:"industry"`
	Keywords  string `json:"keywords"  csv:"keywords"`
	Links     string `json:"links"     csv:"links"`
	Email     string `json:"email"     csv:"email"`
	Phone     string `json:"phone"     csv:"phone"`
}

// Account represents an apollo.io user account.
type Account struct {
	Email         string `json:"email"          csv:"email"`
	Password      string `json:"password"       csv:"password"`
	URL           string `json:"url"            csv:"url"`
	List          string `json:"list"           csv:"list"`
	VPNFile       string `json:"vpn-file"       csv:"vpn-file"`
	Saved         int    `json:"saved"          csv:"saved"`
	Target        int    `json:"target"         csv:"target"`
	Credits       int    `json:"credits"        csv:"credits"`
	CreditRefresh *Time  `json:"credit-refresh" csv:"credit-refresh"`
	loginCookies  []*proto.NetworkCookie
}

func (a *Account) CheckCookieValidity() bool {
	if len(a.loginCookies) == 0 {
		return false
	}

	for _, cookie := range a.loginCookies {
		expiry := cookie.Expires.Time()
		if time.Now().After(cookie.Expires.Time()) && expiry.Year() != 1970 {
			return false
		}
	}

	return true
}

// CanScrape returns true if an [*Account] has enough credits to continue
// scraping leads.
func (a *Account) CanScrape() bool {
	return a.Credits > 0
}

func (a *Account) GetLoginCookies() ([]*proto.NetworkCookie, bool) {
	return a.loginCookies, len(a.loginCookies) > 0
}

// Increment increases the amount of leads saved by a specified amount.
func (a *Account) Increment(amount int) {
	a.Saved += amount
}

// IsDone returns true if the [*Account] has saved the target number of leads.
func (a *Account) IsDone() bool {
	return a.Target == a.Saved
}

func (a *Account) SetLoginCookies(cookies []*proto.NetworkCookie) {
	a.loginCookies = cookies
}

// UseCredits decreases the amount of credits available by a specified amount.
func (a *Account) UseCredits(amount int) {
	a.Credits -= amount
}
