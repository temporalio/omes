# Configuration for OMES worker and executor scripts
# Edit the values below for your environment

set -x cell "s-act-alex-19"
set -x ns "omes-sch-18"
set -x runid "sch_load1"
set -x scenario "scheduler_stress"
set -x omes_image_tag "b973846-go-1.37.0"

# ECR registry for OMES images
set -x omes_ecr_registry "450777629615.dkr.ecr.us-west-2.amazonaws.com/omes"
