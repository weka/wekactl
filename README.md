
# wekactl

>  Open source utility for managing weka cloud formation stacks

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

## Additional info
Importing weka cluster will create the following resources:
- **KMS key**
- **DynamoDB** table: stores weka cluster username and password using the KMS key
- for both backends and clients:
    - **Lambda**:
      - for Api Gateway:
        - *join* 
      - for State Machine: 
        - *fetch*
        - *scale*
        - *terminate*
        - *transient*
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
