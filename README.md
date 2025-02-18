<!--
SPDX-License-Identifier: MPL-2.0
-->

# uks-controller

A custom Kubernetes controller that provides seamless integration with the UpCloud API, allowing you to manage UpCloud Virtual Machines directly through `kubectl`.

## Features

- View and monitor UpCloud VMs using standard Kubernetes tooling
- Prometheus metrics integration for monitoring sync status
- Configurable sync interval
- Detailed logging with multiple verbosity levels

## Quickstart

### Pull vendored `upcloud-go-api`

```shell
git submodule update --recursive
```

### Build and run it locally

```shell
make install run
```

## Configuration

Syncing interval can be configured by `--sync-interval` parameter when the controller is started. By default, VMs are being synced every 15 seconds.


## Implementation

Number of currently syncing UpCloud VMs are available via Prometheus interface (metric name: `virtual_machine_syncing_vms`).

## Further development

  - [ ] Unit testing
  - [ ] Kubernetes events
  - [ ] Enable metrics protection

## Logging

  - `V=1` - we want always see it
  - `V=2` - a bit more verbosity
  - `V=3` - we are debugging the controller, loads of big structures shall we logged

The logging itself is controller by `-zap-log-level` argument, e.g. `-zap-log-level=3` gives you plenty of information on state changes.

## Assumptions

UpCloud API credentials are set in `os.Environ`

## Observations

- Creating _Machine API Key_ isn't straight-forward
- Creating a sub-user
  - **Create subaccount**
    - is grayed out without **any explanation** unless free trial is started (shall indicate what's wrong)
- Having `UpCloudLtd` as a github org name makes a path like this `/go/pkg/mod/github.com/!up!cloud!ltd`. Those `!` make life a bit worse. Try to _cd_ to this directory. And do it again.


## References

  - https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/
  - https://github.com/UpCloudLtd/upcloud-go-api
  - https://developers.upcloud.com/1.3/
  - https://upcloud.com/resources/tutorials/getting-started-upcloud-api
  - https://upcloud.com/docs/getting-started/free-trial/
  - https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md#what-method-to-use
  - https://book.kubebuilder.io/
  - https://go.dev/doc/modules/gomod-ref

## Licence

Licensed under the [Mozilla Public License 2.0 (MPL-2.0)](https://www.mozilla.org/en-US/MPL/2.0/).
See [LICENCE](LICENSE.md) for details and [legal TL;DR](https://www.tldrlegal.com/license/mozilla-public-license-2-0-mpl-2).
