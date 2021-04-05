# wekactl

The wekactl utility is an open-source utility for managing Weka CloudFormation stacks.

The utility can import Weka CloudFormation stacks and manage it via AWS Auto Scaling groups, allowing to scale the Weka cluster up and down.

# Requirements

- For initial setup, the user running the wekactl utility should have AWS admin credentials.
- All the resources created by the utility are made on the AWS account that contains a Weka cluster.
- Only clusters with internet connectivity or those explicitly configured to communicate with an API gateway via a proxy are supported. For new instances to join the Weka cluster, they need to communicate with an API gateway to fetch the cluster information.

## Basic Usage

- Make sure you have access to the AWS account where the Weka cluster resides.
- Download the wekactl binary from the [latest release](https://github.com/weka/wekactl/releases/latest).
- chmod +x PATH_TO_WEKACTL_BINARY

### Importing a Weka Cluster (admin credentials required)

```
PATH_TO_WEKACTL_BINARY cluster import -n CLUSTER_NAME -u WEKA_USERNAME -p WEKA_PASSWORD --region CLUSTER_REGION
```


### Destroying an existing cluster

```
PATH_TO_WEKACTL_BINARY cluster destroy -n CLUSTER_NAME --region CLUSTER_REGION
```

**--keep-instances**: for keeping auto-scaling group instances.

*Note: the cloud formation stack will not be deleted. i.e., destroy removes only the resources created by the wekactl utility.*

### Changing cluster credentials
    PATH_TO_WEKACTL_BINARY cluster change-credentials -n CLUSTER_NAME  -u NEW_WEKA_USERNAME -p NEW_WEKA_PASSWORD --region CLUSTER_REGION

### Notes

- Unhealthy instances, as identified by Weka: instances with user-invoked drives deactivate or stopped weka containers considered as unhealthy by Weka and will be removed from the Weka cluster and replaced with new instances.
- Filesystem scaling is not supported. For scaling down, the filesystems must be in a size that can fit into the shrunk cluster. Alternatively, tiering to S3 can be used to allow downscaling. Future weka versions will address that.

## Additional info

Importing a Weka cluster will create the following resources:

- **KMS key**

- **DynamoDB** table (stores the Weka cluster username and password using the KMS key)

- For both backends and clients:

- - **Lambda**:

  - - for API Gateway:

    - - *join* - responsible for providing cluster information to new instances

    - for State Machine:

    - - *fetch* - fetches cluster/autoscaling group information and passes to the next stage
      - *scale* - relied on *fetch* information to work on the Weka cluster, i.e., deactivate drives/hosts. Will fail if the required target is not supported (like scaling down to 2 backend instances)
      - *terminate* - terminates deactivated hosts
      - *transient* - lambda responsible for reporting transient errors, e.g., could not deactivate specific hosts, but some have been deactivated, and the whole flow proceeded

  - **API Gateway**: invokes the *join* lambda function using an API key

  - **Launch Template**: used for new auto-scaling group instances; will run the join script on launch.

  - **Auto Scaling Groups**

  - **State Machine**: invokes the *fetch*, scale, terminate, transient

  - - Uses the previous lambda output as input for the following lambda.
    - **CloudWatch**: invokes the state machine every minute

  - **IAM Roles (and policies)**:

  - - Lambda
    - State Machine
    - Cloud Watch