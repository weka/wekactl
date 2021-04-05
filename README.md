
# wekactl

Open source utility for managing weka cloud formation stacks  
The utility allows importing CloudFormation weka stacks and managing it via AWS Auto Scaling groups, allowing to scale cluster up and down.

# Requirements
- For initial setup AWS admin credentials are required for user of `wekactl`
- All resources are created on account that contains a weka cluster
- Only clusters connected to the internet, or explicitly configured to communicate with API gateway via proxy are supported right now. In order for new instances to join the cluster they need to communicate with API gateway to fetch cluster information

## Basic Usage
- make sure you have access to weka cluster aws account
- download wekactl binary from the latest release: [latest release](https://github.com/weka/wekactl/releases/latest)
- chmod +x PATH_TO_WEKACTL_BINARY

### Importing cluster (Admin credentials required)
    PATH_TO_WEKACTL_BINARY cluster import -n CLUSTER_NAME -u WEKA_USERNAME -p WEKA_PASSWORD --region CLUSTER_REGION

### Destroying existing cluster 
    PATH_TO_WEKACTL_BINARY cluster destroy -n CLUSTER_NAME --region CLUSTER_REGION
**--keep-instances** : for keeping auto scaling group instances <br>
*notice: cloud formation stack will not be deleted. i.e. `destroy` removes the resources created by wekactl.*

### Changing cluster credentials
    PATH_TO_WEKACTL_BINARY cluster change-credentials -n CLUSTER_NAME  -u NEW_WEKA_USERNAME -p NEW_WEKA_PASSWORD --region CLUSTER_REGION

### Notes
- Uhealthy instances, instances with user-invoked drives deactivate or stopped weka containers considered unhealthy and will be removed from cluster and replaced with fresh instances
- Filesystem scaling is not supported right now. For scaling down filesystem must be the size that can fit into shrinked cluster, alternatively, object store tiering can be used to allow downscaling. In future weka versions we will support automatic scaling of filesystems

## Additional info
Importing weka cluster will create the following resources:
- **KMS key**
- **DynamoDB** table: stores weka cluster username and password using the KMS key
- for both backends and clients:
    - **Lambda**:
      - for Api Gateway:
        - *join*  - responsible for providing cluster information to new instances
      - for State Machine: 
        - *fetch* - fetches cluster/autoscaling group information and passes to next stage
        - *scale* - relied on `fetch` information to work on cluster, i.e deactivate drives/hosts. Will fail if required target is not supported (like scaling down to 2 backend instances)
        - *terminate* - terminates deactivated hosts
        - *transient* - lambda responsible for reporting transient errors, e.g could not deactivate specific hosts, but some were deactivated and whole flow proceeded
    - **API Gateway**: invokes `join` lambda function using api key
    - **Launch Template**: used for new auto scaling group instances,<br>
    will run the `join` script on launch.
    - **Auto Scaling Groups**
    - **State Machine**: invokes `fetch`, `scale`, `terminate`, `transient` <br>
      using previous lambda output as input for the next lambda. 
    - **CloudWatch**: invokes the state machine evey minute
    - **IAM Roles (and policies)**:
      - Lambda
      - State Machine
      - Cloud Watch
