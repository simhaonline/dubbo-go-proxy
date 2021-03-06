/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import (
	"errors"
	"fmt"
)

import (
	"github.com/dubbogo/dubbo-go-proxy/pkg/common/constant"
	"github.com/dubbogo/dubbo-go-proxy/pkg/common/extension"
	"github.com/dubbogo/dubbo-go-proxy/pkg/config"
	"github.com/dubbogo/dubbo-go-proxy/pkg/router"
	"github.com/dubbogo/dubbo-go-proxy/pkg/service"
	"strings"
)

func init() {
	extension.SetAPIDiscoveryService(constant.LocalMemoryApiDiscoveryService, NewLocalMemoryAPIDiscoveryService())
}

// LocalMemoryAPIDiscoveryService is the local cached API discovery service
type LocalMemoryAPIDiscoveryService struct {
	router *router.Route
}

// NewLocalMemoryAPIDiscoveryService creates a new LocalMemoryApiDiscoveryService instance
func NewLocalMemoryAPIDiscoveryService() *LocalMemoryAPIDiscoveryService {
	return &LocalMemoryAPIDiscoveryService{
		router: router.NewRoute(),
	}
}

// AddAPI adds a method to the router tree
func (ads *LocalMemoryAPIDiscoveryService) AddAPI(api router.API) error {
	return ads.router.PutAPI(api)
}

// GetAPI returns the method to the caller
func (ads *LocalMemoryAPIDiscoveryService) GetAPI(url string, httpVerb config.HTTPVerb) (router.API, error) {
	if api, ok := ads.router.FindAPI(url, httpVerb); ok {
		return *api, nil
	}

	return router.API{}, errors.New("not found")
}

// InitAPIsFromConfig inits the router from API config and to local cache
func InitAPIsFromConfig(apiConfig config.APIConfig) error {
	localAPIDiscSrv := extension.GetMustAPIDiscoveryService(constant.LocalMemoryApiDiscoveryService)
	if len(apiConfig.Resources) == 0 {
		return nil
	}
	return loadAPIFromResource("", apiConfig.Resources, localAPIDiscSrv)
}

func loadAPIFromResource(parrentPath string, resources []config.Resource, localSrv service.APIDiscoveryService) error {
	errStack := []string{}
	if len(resources) == 0 {
		return nil
	}
	groupPath := parrentPath
	if parrentPath == constant.PathSlash {
		groupPath = ""
	}
	for _, resource := range resources {
		fullPath := groupPath + resource.Path
		if !strings.HasPrefix(resource.Path, constant.PathSlash) {
			errStack = append(errStack, fmt.Sprintf("Path %s in %s doesn't start with /", resource.Path, parrentPath))
			continue
		}
		if len(resource.Resources) > 0 {
			if err := loadAPIFromResource(resource.Path, resource.Resources, localSrv); err != nil {
				errStack = append(errStack, err.Error())
			}
		}

		if err := loadAPIFromMethods(fullPath, resource.Methods, localSrv); err != nil {
			errStack = append(errStack, err.Error())
		}
	}
	if len(errStack) > 0 {
		return errors.New(strings.Join(errStack, "; "))
	}
	return nil
}

func loadAPIFromMethods(fullPath string, methods []config.Method, localSrv service.APIDiscoveryService) error {
	errStack := []string{}
	for _, method := range methods {
		api := router.API{
			URLPattern: fullPath,
			Method:     method,
		}
		if err := localSrv.AddAPI(api); err != nil {
			errStack = append(errStack, fmt.Sprintf("Path: %s, Method: %s, error: %s", fullPath, method.HTTPVerb, err.Error()))
		}
	}
	if len(errStack) > 0 {
		return errors.New(strings.Join(errStack, "\n"))
	}
	return nil
}
