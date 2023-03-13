ECR setup for docker push

1. Download aws
* should be supported in any package manager (bash, apt)

2. Download aws2: 
* curl 'https://d1vvhvl2y92vvt.cloudfront.net/awscli-exe-macos.zip' -o 'awscli-exe.zip'
* unzip awscli-exe.zip
* sudo ./aws/install

3. Configure aws2:
* aws2 configure sso (will output a profile name)
	* https://d-9d670761e1.awsapps.com/start
    * ca-central-1
* `aws ecr-public get-login-password --region us-east-1 --profile <profile name> | docker login --username AWS —password-stdin public.ecr.aws`

4. Create image repository on ECR
* `aws ecr-public create-repository --repository-name <image name> --region us-east-1 --profile <profile name>`

5. Push docker image
* bring image to ECR format: `docker tag <image name> public.ecr.aws/p5q2r9h7/<image name>`
* `docker push public.ecr.aws/p5q2r9h7/<image name>`