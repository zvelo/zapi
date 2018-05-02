package zmage

import (
	"strings"
	"time"
)

func buildTag(image string) (string, error) {
	if i := strings.Index(image, ":"); i != -1 {
		return image[i+1:], nil
	}

	b, err := branch()
	if err != nil {
		return "", err
	}

	if b == "master" {
		return "latest", nil
	}

	return b, nil
}

func buildArgs() ([]string, error) {
	v, err := version()
	if err != nil {
		return nil, err
	}

	hash, err := commitHash()
	if err != nil {
		return nil, err
	}

	return []string{
		"--build-arg", "VERSION=" + v,
		"--build-arg", "GIT_COMMIT=" + hash,
	}, nil
}

func BuildImage(image, dockerFile, ctxDir string) error {
	tag, err := buildTag(image)
	if err != nil {
		return err
	}

	ba, err := buildArgs()
	if err != nil {
		return err
	}

	args := []string{
		"build",
		"-t", image + ":" + tag,
		"-f", dockerFile,
	}
	args = append(args, ba...)
	args = append(args, ctxDir)

	return docker(args...)
}

func PushImage(image string) error {
	bt, err := buildTag(image)
	if err != nil {
		return err
	}

	var tags []string

	if bt == "latest" {
		var v string
		if v, err = version(); err != nil {
			return err
		}

		if err = docker("push", image+":latest"); err != nil {
			return err
		}

		tags = append(tags, v)
	} else {
		var ch string
		if ch, err = commitHash(); err != nil {
			return err
		}

		tags = append(tags, time.Now().Format("20060102-151405")+"-"+ch)

		var b string
		if b, err = branch(); err != nil {
			return err
		}

		tags = append(tags, b)
	}

	for _, tag := range tags {
		if err = docker("tag", image+":"+bt, image+":"+tag); err != nil {
			return err
		}

		if err = docker("push", image+":"+tag); err != nil {
			return err
		}
	}

	return nil
}
