# wekactl

The wekactl utility is an open-source utility for managing Weka CloudFormation stacks.

The utility can import Weka CloudFormation stacks and manage it via AWS Auto Scaling groups, allowing to scale the Weka cluster up and down.

Once deployed, you can control the number of instances by either changing the desired capacity of instances from the AWS auto-scaling group console or defining your custom metrics and scaling policy in AWS. Once the desired capacity has changed, Weka will take care of safely scaling the instances.

# Requirements

- For initial setup, the user running the wekactl utility should have AWS admin credentials.
- All the resources created by the utility are made on the AWS account that contains a Weka cluster.

### Deploying in Private Networks
- By default, clusters are expected to have internet connectivity or be explicitly configured to communicate with an API gateway via a proxy.
When deployed in a private VPC without the above connectivity, you can use the `--private-subnet` flag to instructs wekactl to create resources in a private subnet, i.e., without allocating public IP addresses and without deploying an API gateway inside the VPC.
- Before running wekactl in a private VPC, make sure that:
  - The VPC of the cluster has a VPC endpoint to execute-api.
  - The VPC endpoint should be open via a security group to the cluster security groups (or the whole VPC)
You can follow the instruction on the [Create an interface VPC endpoint for API Gateway execute-api](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-private-apis.html#apigateway-private-api-create-interface-vpc-endpoint "AWS Documentation") section on AWS documentation.


## Basic Usage

- Make sure you have access to the AWS account where the Weka cluster resides.
- Download the wekactl binary from the [latest release](https://github.com/weka/wekactl/releases/latest).
- chmod +x PATH_TO_WEKACTL_BINARY

### Importing a Weka Cluster (admin credentials required)

```
PATH_TO_WEKACTL_BINARY cluster import -n CLUSTER_NAME -u WEKA_USERNAME -p WEKA_PASSWORD --region CLUSTER_REGION
```

The import reads the CloudFormation stack and creates Auto Scaling Groups based on that.
It Creates an auto-scaling group for the Weka cluster (backend servers), and if clients have been deployed via the CloudFormation stack, it also creates an auto-scaling group for the clients.

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
- When scaling-in, the Auto-Scaling Group activity log will show the operation as Cancelled since Weka protects the instances from un-managed and us-safe scale-in. Yet, the wekactl utility receives the scale-in request and manages the scale-in. Once the managed deactivation of hosts and drives from the cluster finishes, the wekactl utility removes the scale-in protection. Then, the instances are terminated and removed from the Auto-Scaling group (will be seen on the Auto-Scaling Group activity log).
- As noted, the wekactl utility can manage client instances auto-scaling, also replacing unhealthy client instances. If that is not desirable for clients, remove the auto-scaling group generated for the clients.

## Additional info

Importing a Weka cluster will create the following resources (note, these should not be exposed to a user that has not ClusterAdmin permission for the Weka cluster):

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
