/*
 Copyright 2022 The KubeSphere Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package images

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
)

// 定义结构体来匹配 JSON 结构
type ImageManifest struct {
	Manifests     []ManifestEntry `json:"manifests"`
	MediaType     string          `json:"mediaType"`
	SchemaVersion int             `json:"schemaVersion"`
}

type CopyImageOptions struct {
	srcImage  *srcImageOptions
	destImage *destImageOptions
}

type ManifestEntry struct {
	Digest    string          `json:"digest"`
	MediaType string          `json:"mediaType"`
	Platform  PlatformDetails `json:"platform"`
	Size      int             `json:"size"`
}

type PlatformDetails struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Variant      string `json:"variant,omitempty"` // 用于可选字段
}

func (c *CopyImageOptions) Check() (bool, error) {
	srcContext := c.srcImage.systemContext()
	srcRef, err := alltransports.ParseImageName(c.srcImage.imageName)
	if err != nil {
		return false, err
	}
	ctx := context.Background()

	src, err := srcRef.NewImageSource(ctx, srcContext)
	if err != nil {
		return false, err
	}

	// 获取镜像的清单数据和 MIME 类型
	manifestData, _, err := src.GetManifest(ctx, nil)
	if err != nil {
		return false, err
	}

	// 解析 JSON 数据
	var manifest ImageManifest
	if err := json.Unmarshal([]byte(string(manifestData)), &manifest); err != nil {
		return false, err
	}

	// 获取并打印每个 manifest 的 architecture 值
	if manifest.Manifests == nil {
		// 解析镜像引用
		ref, err := docker.ParseReference("//" + strings.Split(c.srcImage.imageName, "//")[1])
		if err != nil {
			return false, err
		}

		// 获取镜像的详细信息
		img, err := ref.NewImage(ctx, srcContext)
		if err != nil {
			return false, err
		}
		defer img.Close()

		// 使用 Inspect 方法获取镜像信息
		inspectedImage, err := img.Inspect(ctx)
		if err != nil {
			return false, err
		}

		if inspectedImage.Architecture != c.destImage.dockerImage.arch {
			return false, nil
		}
	}

	return true, nil
}
func (c *CopyImageOptions) Copy() error {
	policyContext, err := getPolicyContext()
	if err != nil {
		return err
	}
	defer policyContext.Destroy()

	srcRef, err := alltransports.ParseImageName(c.srcImage.imageName)
	if err != nil {
		return err
	}
	destRef, err := alltransports.ParseImageName(c.destImage.imageName)
	if err != nil {
		return err
	}

	srcContext := c.srcImage.systemContext()
	destContext := c.destImage.systemContext()

	_, err = copy.Image(context.Background(), policyContext, destRef, srcRef, &copy.Options{
		ReportWriter:   os.Stdout,
		SourceCtx:      srcContext,
		DestinationCtx: destContext,
	})
	if err != nil {
		return err
	}
	return nil
}

func getPolicyContext() (*signature.PolicyContext, error) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	return signature.NewPolicyContext(policy)
}

type Index struct {
	Manifests []Manifest
}

type Manifest struct {
	Annotations annotations
}

type annotations struct {
	RefName string `json:"org.opencontainers.image.ref.name"`
}

func NewIndex() *Index {
	return &Index{
		Manifests: []Manifest{},
	}
}
