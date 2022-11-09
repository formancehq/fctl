package internal

import (
	"fmt"
	"net/url"
)

func ServicesBaseUrl(profile Profile, organization, stack string) (*url.URL, error) {
	baseUrl, err := url.Parse(profile.baseServiceURI)
	if err != nil {
		return nil, err
	}
	baseUrl.Host = fmt.Sprintf("%s-%s.%s", organization, stack, baseUrl.Host)
	return baseUrl, nil
}

func ApiUrl(profile Profile, organization, stack, service string) (*url.URL, error) {
	url, err := ServicesBaseUrl(profile, organization, stack)
	if err != nil {
		return nil, err
	}
	url.Path = "/api/" + service
	return url, nil
}

func MustApiUrl(profile Profile, organization, stack, service string) *url.URL {
	url, err := ApiUrl(profile, organization, stack, service)
	if err != nil {
		panic(err)
	}
	return url
}
