package process

import (
	"bytes"
	"github.com/daniel-cole/phunter/system"
	"github.com/sirupsen/logrus"
	"os/exec"
	"strings"
	"text/template"
)

var bashCmd = `
container_pid=[[[ . ]]]
container_pid_ns=$(lsns -no NS -t pid -p "${container_pid}")
found=false
for UUID in $(docker ps -q); do
 pid=$(docker inspect -f '{{.State.Pid}}' "$UUID")
 name=$(docker inspect -f '{{.Name}}' "$UUID")
 name=${name#/}

 pidns=$(stat --format="%N" /proc/"${pid}"/ns/pid)
 pidns=${pidns#*[}
 pidns=${pidns%]*}

 if [[ "${container_pid_ns}" -eq "${pidns}" ]]; then
   echo "${name}"
   found=true
   break
 fi
done

if ! $found; then
 >&2 echo "unable to find container for pid: ${container_pid}"
 exit 1
fi
`

// FindContainerName returns the container name that the specified process is running in
// it uses the inline bash script provided above as a template to execute on the system
func (p *Process) FindContainerName() (string, error) {
	logrus.WithField("pid", p.ID).Debugf("attempting to find container name by ID")
	cmd := exec.Command("bash", "-s")
	cmdTmpl, err := template.New("get_container_name_by_pid.sh").Delims("[[[", "]]]").Parse(bashCmd)
	if err != nil {
		logrus.WithField("pid", p.ID).Errorf("failed to render template: %v\n", err)
		return "", err
	}

	var renderedCmd bytes.Buffer
	err = cmdTmpl.Execute(&renderedCmd, p.ID)
	if err != nil {
		logrus.WithField("pid", p.ID).Errorf("failed to render template for finding container: %v", err)
		return "", err
	}
	cmd.Stdin = strings.NewReader(string(renderedCmd.Bytes()))

	containerName, err := system.RunCmd(cmd)
	if err != nil {
		logrus.WithField("pid", p.ID).Errorf("failed to get container name, %v", err)
		return "", err
	}
	return strings.TrimRight(containerName, "\n"), nil
}
