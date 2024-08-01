package certificates

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"

	capi "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	certificatesinformers "k8s.io/client-go/informers/certificates/v1"
	clientset "k8s.io/client-go/kubernetes"
	certificateslisters "k8s.io/client-go/listers/certificates/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type CertificateController struct {
	name       string
	kubeClient clientset.Interface
	csrLister  certificateslisters.CertificateSigningRequestLister
	csrsSynced cache.InformerSynced
	handler    func(context.Context, *capi.CertificateSigningRequest) error
	queue      workqueue.RateLimitingInterface
}

func NewCertificateController(
	ctx context.Context,
	name string,
	kubeClient clientset.Interface,
	csrInformer certificatesinformers.CertificateSigningRequestInformer,
	handler func(context.Context, *capi.CertificateSigningRequest) error,
) *CertificateController {
	logger := klog.FromContext(ctx)
	cc := &CertificateController{
		name:       name,
		kubeClient: kubeClient,
		queue: workqueue.NewRateLimitingQueueWithConfig(workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(200*time.Millisecond, 1000*time.Second),
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		), workqueue.RateLimitingQueueConfig{Name: "certificate"}),
		handler: handler,
	}

	_, err := csrInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			csr := obj.(*capi.CertificateSigningRequest)
			logger.V(4).Info("Adding certificate request", "csr", csr.Name)
			cc.enqueueCertificateRequest(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			oldCSR := old.(*capi.CertificateSigningRequest)
			logger.V(4).Info("Updating certificate request", "old", oldCSR.Name)
			cc.enqueueCertificateRequest(new)
		},
		DeleteFunc: func(obj interface{}) {
			csr, ok := obj.(*capi.CertificateSigningRequest)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					logger.V(2).Info("Couldn't get object from tombstone", "object", obj)
					return
				}
				csr, ok = tombstone.Obj.(*capi.CertificateSigningRequest)
				if !ok {
					logger.V(2).Info("Tombstone contained object that is not a CSR", "object", obj)
					return
				}
			}
			logger.V(4).Info("Deleting certificate request", "csr", csr.Name)
			cc.enqueueCertificateRequest(obj)
		},
	})
	if err != nil {
		klog.Exitf("Error adding certificate controller event handler: %v", err)
	}

	cc.csrLister = csrInformer.Lister()
	cc.csrsSynced = csrInformer.Informer().HasSynced

	return cc
}

func (cc *CertificateController) Run(ctx context.Context, workers int) {
	defer utilruntime.HandleCrash()
	defer cc.queue.ShutDown()

	logger := klog.FromContext(ctx)
	logger.Info("Starting certificate controller", "name", cc.name)
	defer logger.Info("Shutting down certificate controller", "name", cc.name)

	if !cache.WaitForNamedCacheSync(fmt.Sprintf("certificate-%s", cc.name), ctx.Done(), cc.csrsSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, cc.worker, time.Second)
	}

	<-ctx.Done()
}

func (cc *CertificateController) worker(ctx context.Context) {
	for cc.processNextWorkItem(ctx) {
	}
}

func (cc *CertificateController) processNextWorkItem(ctx context.Context) bool {
	cKey, quit := cc.queue.Get()
	if quit {
		return false
	}
	defer cc.queue.Done(cKey)

	if err := cc.syncFunc(ctx, cKey.(string)); err != nil {
		cc.queue.AddRateLimited(cKey)
		if _, ignorable := err.(ignorableError); !ignorable {
			utilruntime.HandleError(fmt.Errorf("sync %v failed with: %v", cKey, err))
		} else {
			klog.FromContext(ctx).V(4).Info("sync certificate request failed", "csr", cKey, "err", err)
		}
		return true
	}

	cc.queue.Forget(cKey)
	return true

}

func (cc *CertificateController) enqueueCertificateRequest(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %+v: %v", obj, err))
		return
	}
	cc.queue.Add(key)
}

func (cc *CertificateController) syncFunc(ctx context.Context, key string) error {
	logger := klog.FromContext(ctx)
	startTime := time.Now()
	defer func() {
		logger.V(4).Info("Finished syncing certificate request", "csr", key, "elapsedTime", time.Since(startTime))
	}()
	csr, err := cc.csrLister.Get(key)
	if errors.IsNotFound(err) {
		logger.V(3).Info("csr has been deleted", "csr", key)
		return nil
	}
	if err != nil {
		return err
	}

	if len(csr.Status.Certificate) > 0 {
		// no need to do anything because it already has a cert
		return nil
	}

	// need to operate on a copy so we don't mutate the csr in the shared cache
	csr = csr.DeepCopy()
	return cc.handler(ctx, csr)
}

func IgnorableError(s string, args ...interface{}) ignorableError {
	return ignorableError(fmt.Sprintf(s, args...))
}

type ignorableError string

func (e ignorableError) Error() string {
	return string(e)
}
