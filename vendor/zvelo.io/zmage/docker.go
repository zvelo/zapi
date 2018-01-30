package zmage

import "time"

func BuildImage(image, dockerFile, ctxDir string) error {
	return docker("build", "-t", image, "-f", dockerFile, ctxDir)
}

func PushImage(image string) error {
	b, err := branch()
	if err != nil {
		return err
	}

	if b == "master" {
		var v string
		if v, err = version(); err != nil {
			return err
		}

		if err = docker("push", image+":latest"); err != nil {
			return err
		}

		if err = docker("tag", image+":latest", image+":"+v); err != nil {
			return err
		}

		return docker("push", image+":"+v)
	}

	ch, err := commitHash()
	if err != nil {
		return err
	}

	imageTag := time.Now().Format("20060102-151405") + "-" + ch

	if err = docker("tag", image+":latest", image+":"+imageTag); err != nil {
		return err
	}

	return docker("push", image+":"+imageTag)
}
