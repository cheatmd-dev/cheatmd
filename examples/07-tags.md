---
tags: [demo, reference]
---

# 07 · Tags

Tags fold into picker search. A cheat collects them from five sources: the
folder/file path, YAML front matter (above, file-wide), a footer block (below,
file-wide), inline `#tag` in prose (per cheat), and words in the heading.

## List buckets (s3)

Lists all S3 buckets in the account. #read-only

```sh title:"List S3 buckets"
echo "aws s3 ls"
```

#cloud

## Describe instance

Inline tags above attach only to "List buckets"; they do not leak here.

```sh title:"Describe an EC2 instance"
echo "aws ec2 describe-instances --instance-id $id"
```
<!-- cheat
var id --- --header "Instance ID"
-->

<!-- Footer hashtags below are file-wide, like the front matter above. -->

#quickref #internal
