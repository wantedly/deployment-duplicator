package testing

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/stuart-warren/yamlfmt"
)

func SnapshotYaml(t *testing.T, objs ...interface{}) {
	t.Helper()

	manifests := make([]string, len(objs))

	for i, obj := range objs {

		// struct to map
		rs := make(map[string]interface{})
		{ // Marshal into json string to omit unused fields
			jsnStr, err := json.Marshal(obj)
			if err != nil {
				t.Fatal(err)
			}
			err = json.Unmarshal(jsnStr, &rs)
			if err != nil {
				t.Fatal(err)
			}
		}

		// map to formatted yaml
		var formatted string
		{
			d, err := yaml.Marshal(&rs)
			if err != nil {
				t.Fatal(err)
			}

			formatted, err = format(d)
			if err != nil {
				t.Fatal(err)
			}
		}
		manifests[i] = formatted
	}

	recorder := cupaloy.New(cupaloy.SnapshotFileExtension(".yaml"))
	recorder.SnapshotT(t, strings.Join(manifests, "\n"))
}

func format(content []byte) (string, error) {
	bs, err := yamlfmt.Format(bytes.NewReader(content))

	if err != nil {
		return "", errors.Wrap(err, "failed to format yaml")
	}
	return string(bs), nil
}
