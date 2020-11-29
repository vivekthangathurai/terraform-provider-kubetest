package kubetest

import (
	"context"
	"log"
	//"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)
func resourceEvent() *schema.Resource {
	return &schema.Resource{
	  Create: resourceEventCreate,
	  Read:   resourceEventRead,
	  Update: resourceEventUpdate,
	  Delete: resourceEventDelete,
  
	  Schema: map[string]*schema.Schema{
		"image": {
			Type:     schema.TypeString,
			Required: true,
		  },
		"namespace": {
			Type:     schema.TypeString,
			Required: true,
		  }, 
	  },
	}
  }
  
  func resourceEventCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	pod := &api.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					Name:  "test-pod",
					Image: d.Get("image").(string),
				},
			},
		},
	}
	log.Printf("[INFO] Creating new pod: %#v", pod)
	out, err := conn.CoreV1().Pods(d.Get("namespace").(string),).Create(context.Background(), pod, metav1.CreateOptions{})

	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new pod: %#v", out)
	return nil
  }
  
  func resourceEventRead(d *schema.ResourceData, meta interface{}) error {
	// TODO
	return nil
  }
  
  func resourceEventUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO
	return nil
  }
  
  func resourceEventDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO
	return nil
  }