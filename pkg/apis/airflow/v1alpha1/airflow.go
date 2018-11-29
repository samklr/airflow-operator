/*
Copyright 2018 Google LLC
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"bytes"
	"fmt"
	application "github.com/kubernetes-sigs/application/pkg/apis/app/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math/rand"
	"sigs.k8s.io/kubesdk/pkg/finalizer"
	"sigs.k8s.io/kubesdk/pkg/resource"
	"strconv"
	"time"
)

// constants defining field values
const (
	ControllerVersion = "0.1"

	PasswordCharNumSpace = "abcdefghijklmnopqrstuvwxyz0123456789"
	PasswordCharSpace    = "abcdefghijklmnopqrstuvwxyz"

	ActionCheck  = "check"
	ActionCreate = "create"
	ActionDelete = "delete"

	LabelAirflowCR                 = "airflow-cr"
	ValueAirflowCRBase             = "airflow-base"
	ValueAirflowCRCluster          = "airflow-cluster"
	LabelAirflowCRName             = "airflow-cr-name"
	LabelAirflowComponent          = "airflow-component"
	ValueAirflowComponentMySQL     = "mysql"
	ValueAirflowComponentPostgres  = "postgres"
	ValueAirflowComponentSQLProxy  = "sqlproxy"
	ValueAirflowComponentSQL       = "sql"
	ValueAirflowComponentUI        = "airflowui"
	ValueAirflowComponentNFS       = "nfs"
	ValueAirflowComponentRedis     = "redis"
	ValueAirflowComponentScheduler = "scheduler"
	ValueAirflowComponentWorker    = "worker"
	ValueAirflowComponentFlower    = "flower"
	LabelControllerVersion         = "airflow-controller-version"
	LabelApp                       = "app"

	KindAirflowBase    = "AirflowBase"
	KindAirflowCluster = "AirflowCluster"

	PodManagementPolicyParallel = "Parallel"

	GitSyncDestDir  = "gitdags"
	GCSSyncDestDir  = "dags"
	afk             = "AIRFLOW__KUBERNETES__"
	afc             = "AIRFLOW__CORE__"
	AirflowHome     = "/usr/local/airflow"
	AirflowDagsBase = AirflowHome + "/dags/"
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// AirflowResource an interface for operating on AirflowResource
type AirflowResource interface {
	getMeta(name string, labels map[string]string) metav1.ObjectMeta
	getAffinity() *corev1.Affinity
	getNodeSelector() map[string]string
	getName() string
	getLabels() map[string]string
	getCRName() string
}

func optionsToString(options map[string]string, prefix string) string {
	var buf bytes.Buffer
	for k, v := range options {
		buf.WriteString(fmt.Sprintf("%s%s %s ", prefix, k, v))
	}
	return buf.String()
}

// RandomAlphanumericString generates a random password of some fixed length.
func RandomAlphanumericString(strlen int) []byte {
	result := make([]byte, strlen)
	for i := range result {
		result[i] = PasswordCharNumSpace[random.Intn(len(PasswordCharNumSpace))]
	}
	result[0] = PasswordCharSpace[random.Intn(len(PasswordCharSpace))]
	return result
}

func envFromSecret(name string, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Key: key,
		},
	}
}

func rsrcName(name string, component string, suffix string) string {
	return name + "-" + component + suffix
}

func (r *AirflowBase) getName() string { return r.Name }

func (r *AirflowBase) getAffinity() *corev1.Affinity { return r.Spec.Affinity }

func (r *AirflowBase) getNodeSelector() map[string]string { return r.Spec.NodeSelector }

func (r *AirflowBase) getLabels() map[string]string { return r.Spec.Labels }

func (r *AirflowBase) getCRName() string { return ValueAirflowCRBase }

func (r *AirflowBase) getMeta(name string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:   r.Namespace,
		Annotations: r.Spec.Annotations,
		Labels:      labels,
		Name:        name,
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(r, schema.GroupVersionKind{
				Group:   SchemeGroupVersion.Group,
				Version: SchemeGroupVersion.Version,
				Kind:    KindAirflowBase,
			}),
		},
	}
}

func (r *AirflowCluster) getName() string { return r.Name }

func (r *AirflowCluster) getAffinity() *corev1.Affinity { return r.Spec.Affinity }

func (r *AirflowCluster) getNodeSelector() map[string]string { return r.Spec.NodeSelector }

func (r *AirflowCluster) getLabels() map[string]string { return r.Spec.Labels }

func (r *AirflowCluster) getCRName() string { return ValueAirflowCRCluster }

func (r *AirflowCluster) getMeta(name string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:   r.Namespace,
		Annotations: r.Spec.Annotations,
		Labels:      labels,
		Name:        name,
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(r, schema.GroupVersionKind{
				Group:   SchemeGroupVersion.Group,
				Version: SchemeGroupVersion.Version,
				Kind:    KindAirflowCluster,
			}),
		},
	}
}

func getApplicationFakeRemove() *application.Application {
	return &application.Application{}
}

func (r *AirflowCluster) getAirflowPrometheusEnv() []corev1.EnvVar {
	sqlSvcName := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	ap := "AIRFLOW_PROMETHEUS_"
	apd := ap + "DATABASE_"
	env := []corev1.EnvVar{
		{Name: ap + "LISTEN_ADDR", Value: ":9112"},
		{Name: apd + "BACKEND", Value: "mysql"},
		{Name: apd + "HOST", Value: sqlSvcName},
		{Name: apd + "PORT", Value: "3306"},
		{Name: apd + "USER", Value: r.Spec.Scheduler.DBUser},
		{Name: apd + "PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: apd + "NAME", Value: r.Spec.Scheduler.DBName},
	}
	return env
}

func (r *AirflowCluster) getAirflowEnv(saName string) []corev1.EnvVar {
	sp := r.Spec
	sqlSvcName := rsrcName(sp.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	redisSecret := rsrcName(r.Name, ValueAirflowComponentRedis, "")
	redisSvcName := redisSecret
	dagFolder := AirflowDagsBase
	if sp.DAGs != nil {
		if sp.DAGs.Git != nil {
			dagFolder = AirflowDagsBase + "/" + GitSyncDestDir + "/" + sp.DAGs.DagSubdir
		} else if sp.DAGs.GCS != nil {
			dagFolder = AirflowDagsBase + "/" + GCSSyncDestDir + "/" + sp.DAGs.DagSubdir
		}
	}
	dbType := "mysql"
	// TODO dbType = "postgres"
	env := []corev1.EnvVar{
		{Name: "EXECUTOR", Value: sp.Executor},
		{Name: "SQL_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: afc + "DAGS_FOLDER", Value: dagFolder},
		{Name: "SQL_HOST", Value: sqlSvcName},
		{Name: "SQL_USER", Value: sp.Scheduler.DBUser},
		{Name: "SQL_DB", Value: sp.Scheduler.DBName},
		{Name: "DB_TYPE", Value: dbType},
	}
	if sp.Executor == ExecutorK8s {
		env = append(env, []corev1.EnvVar{
			{Name: afk + "WORKER_CONTAINER_REPOSITORY", Value: sp.Worker.Image},
			{Name: afk + "WORKER_CONTAINER_TAG", Value: sp.Worker.Version},
			{Name: afk + "WORKER_CONTAINER_IMAGE_PULL_POLICY", Value: "IfNotPresent"},
			{Name: afk + "DELETE_WORKER_PODS", Value: "True"},
			{Name: afk + "NAMESPACE", Value: r.Namespace},
			//{Name: afk+"AIRFLOW_CONFIGMAP", Value: ??},
			//{Name: afk+"IMAGE_PULL_SECRETS", Value: s.ImagePullSecrets},
			//{Name: afk+"GCP_SERVICE_ACCOUNT_KEYS", Vaslue:  ??},
		}...)
		if sp.DAGs != nil && sp.DAGs.Git != nil {
			env = append(env, []corev1.EnvVar{
				{Name: afk + "GIT_REPO", Value: sp.DAGs.Git.Repo},
				{Name: afk + "GIT_BRANCH", Value: sp.DAGs.Git.Branch},
				{Name: afk + "GIT_SUBPATH", Value: sp.DAGs.DagSubdir},
				{Name: afk + "WORKER_SERVICE_ACCOUNT_NAME", Value: saName},
			}...)
			if sp.DAGs.Git.CredSecretRef != nil {
				env = append(env, []corev1.EnvVar{
					{Name: "GIT_PASSWORD",
						ValueFrom: envFromSecret(sp.DAGs.Git.CredSecretRef.Name, "password")},
					{Name: "GIT_USER", Value: sp.DAGs.Git.User},
				}...)
			}
		}
	}
	if sp.Executor == ExecutorCelery {
		env = append(env,
			[]corev1.EnvVar{
				{Name: "REDIS_PASSWORD",
					ValueFrom: envFromSecret(redisSecret, "password")},
				{Name: "REDIS_HOST", Value: redisSvcName},
			}...)
	}
	return env
}

func (r *AirflowCluster) addAirflowContainers(ss *appsv1.StatefulSet, containers []corev1.Container, volName string) {
	ss.Spec.Template.Spec.InitContainers = []corev1.Container{}
	if r.Spec.DAGs != nil {
		init, dagContainer := r.Spec.DAGs.container(volName)
		if init {
			ss.Spec.Template.Spec.InitContainers = []corev1.Container{dagContainer}
		} else {
			containers = append(containers, dagContainer)
		}
	}
	ss.Spec.Template.Spec.Containers = containers
}

func (r *AirflowCluster) addMySQLUserDBContainer(ss *appsv1.StatefulSet) {
	sqlRootSecret := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSvcName := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	env := []corev1.EnvVar{
		{Name: "SQL_ROOT_PASSWORD", ValueFrom: envFromSecret(sqlRootSecret, "rootpassword")},
		{Name: "SQL_DB", Value: r.Spec.Scheduler.DBName},
		{Name: "SQL_USER", Value: r.Spec.Scheduler.DBUser},
		{Name: "SQL_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: "SQL_HOST", Value: sqlSvcName},
		{Name: "DB_TYPE", Value: "mysql"},
	}
	containers := []corev1.Container{
		{
			Name:    "mysql-dbcreate",
			Image:   defaultMySQLImage + ":" + defaultMySQLVersion,
			Env:     env,
			Command: []string{"/bin/bash"},
			//SET GLOBAL explicit_defaults_for_timestamp=ON;
			Args: []string{"-c", `
mysql -uroot -h$(SQL_HOST) -p$(SQL_ROOT_PASSWORD) << EOSQL 
CREATE DATABASE IF NOT EXISTS $(SQL_DB);
USE $(SQL_DB);
CREATE USER IF NOT EXISTS '$(SQL_USER)'@'%' IDENTIFIED BY '$(SQL_PASSWORD)';
GRANT ALL ON $(SQL_DB).* TO '$(SQL_USER)'@'%' ;
FLUSH PRIVILEGES;
EOSQL
`},
		},
	}
	ss.Spec.Template.Spec.InitContainers = append(containers, ss.Spec.Template.Spec.InitContainers...)
}

func (r *AirflowCluster) addPostgresUserDBContainer(ss *appsv1.StatefulSet) {
	sqlRootSecret := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSvcName := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	env := []corev1.EnvVar{
		{Name: "SQL_ROOT_PASSWORD", ValueFrom: envFromSecret(sqlRootSecret, "rootpassword")},
		{Name: "SQL_DB", Value: r.Spec.Scheduler.DBName},
		{Name: "SQL_USER", Value: r.Spec.Scheduler.DBUser},
		{Name: "SQL_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: "SQL_HOST", Value: sqlSvcName},
		{Name: "DB_TYPE", Value: "postgres"},
	}
	containers := []corev1.Container{
		{
			Name:    "postgres-dbcreate",
			Image:   defaultPostgresImage + ":" + defaultPostgresVersion,
			Env:     env,
			Command: []string{"/bin/bash"},
			Args: []string{"-c", `
PGPASSWORD=$(SQL_ROOT_PASSWORD) psql -h $SQL_HOST -U airflow -d testdb -c "CREATE DATABASE $(SQL_DB)";
PGPASSWORD=$(SQL_ROOT_PASSWORD) psql -h $SQL_HOST -U airflow -d testdb -c "CREATE USER $(SQL_USER) WITH ENCRYPTED PASSWORD '$(SQL_PASSWORD)'; GRANT ALL PRIVILEGES ON DATABASE $(SQL_DB) TO $(SQL_USER)"
`},
		},
	}
	ss.Spec.Template.Spec.InitContainers = append(containers, ss.Spec.Template.Spec.InitContainers...)
}

// sts returns a StatefulSet object which specifies
//  CPU and memory
//  resources
//  volume, volume mount
//  pod spec
func sts(r AirflowResource, component string, suffix string, svc bool, labels map[string]string) *appsv1.StatefulSet {
	name := rsrcName(r.getName(), component, suffix)
	svcName := ""
	if svc {
		svcName = name
	}

	meta := r.getMeta(name, labels)
	return &appsv1.StatefulSet{
		ObjectMeta: meta,
		Spec: appsv1.StatefulSetSpec{
			ServiceName: svcName,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: meta.Annotations,
					Labels:      labels,
				},
				Spec: corev1.PodSpec{
					Affinity:     r.getAffinity(),
					NodeSelector: r.getNodeSelector(),
					Subdomain:    name,
				},
			},
		},
	}
}

func service(r AirflowResource, component string, name string, labels map[string]string, ports []corev1.ServicePort) *resource.Object {
	sname := rsrcName(r.getName(), component, "")
	if name == "" {
		name = sname
	}
	return &resource.Object{
		Obj: &corev1.Service{
			ObjectMeta: r.getMeta(name, labels),
			Spec: corev1.ServiceSpec{
				Ports:    ports,
				Selector: labels,
			},
		},
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &corev1.ServiceList{},
	}
}

func podDisruption(r AirflowResource, component string, suffix string, minavail string, labels map[string]string) *resource.Object {
	name := rsrcName(r.getName(), component, suffix)
	minAvailable := intstr.Parse(minavail)

	return &resource.Object{
		Obj: &policyv1.PodDisruptionBudget{
			ObjectMeta: r.getMeta(name, labels),
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvailable,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
			},
		},
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &policyv1.PodDisruptionBudgetList{},
	}
}

// ------------------------------ MYSQL  ---------------------------------------

func (s *MySQLSpec) service(r *AirflowBase, labels map[string]string) *resource.Object {
	return service(r, ValueAirflowComponentMySQL,
		rsrcName(r.Name, ValueAirflowComponentSQL, ""), labels,
		[]corev1.ServicePort{{Name: "mysql", Port: 3306}})
}

func (s *MySQLSpec) podDisruption(r *AirflowBase, labels map[string]string) *resource.Object {
	return podDisruption(r, ValueAirflowComponentMySQL, "", "100%", labels)
}

func (s *MySQLSpec) secret(r *AirflowBase, labels map[string]string) *resource.Object {
	name := rsrcName(r.getName(), ValueAirflowComponentSQL, "")
	return &resource.Object{
		Obj: &corev1.Secret{
			ObjectMeta: r.getMeta(name, labels),
			Data: map[string][]byte{
				"password":     RandomAlphanumericString(16),
				"rootpassword": RandomAlphanumericString(16),
			},
		},
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &corev1.SecretList{},
	}
}

func (s *MySQLSpec) sts(r *AirflowBase, labels map[string]string) *resource.Object {
	sqlSecret := rsrcName(r.getName(), ValueAirflowComponentSQL, "")
	ss := sts(r, ValueAirflowComponentMySQL, "", true, labels)
	ss.Spec.Replicas = &s.Replicas
	volName := "mysql-data"
	if s.VolumeClaimTemplate != nil {
		ss.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*s.VolumeClaimTemplate}
		volName = s.VolumeClaimTemplate.Name
	} else {
		ss.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
	}
	ss.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:  "mysql",
			Image: s.Image + ":" + s.Version,
			Env: []corev1.EnvVar{
				{Name: "MYSQL_DATABASE", Value: "testdb"},
				{Name: "MYSQL_USER", Value: "airflow"},
				{Name: "MYSQL_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
				{Name: "MYSQL_ROOT_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "rootpassword")},
			},
			//Args:      []string{"-c", fmt.Sprintf("exec mysql %s", optionsToString(s.Options, "--"))},
			Args:      []string{"--explicit-defaults-for-timestamp=ON"},
			Resources: s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "mysql",
					ContainerPort: 3306,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/var/lib/mysql",
				},
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"bash", "-c", "mysqladmin -p$MYSQL_ROOT_PASSWORD ping"},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       20,
				TimeoutSeconds:      5,
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"bash", "-c", "mysql -u$MYSQL_USER -p$MYSQL_PASSWORD -e \"use testdb\""},
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       5,
				TimeoutSeconds:      2,
			},
		},
	}
	return &resource.Object{
		Obj:       ss,
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &appsv1.StatefulSetList{},
	}
}

// Mutate - mutate expected
func (s *MySQLSpec) Mutate(rsrc interface{}, status interface{}, expected, observed *resource.ObjectBag) (*resource.ObjectBag, error) {
	return expected, nil
}

// Finalize - execute finalizers
func (s *MySQLSpec) Finalize(rsrc, sts interface{}, observed *resource.ObjectBag) error {
	r := rsrc.(*AirflowBase)
	finalizer.Remove(r, finalizer.Cleanup)
	return nil
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *MySQLSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string) (*resource.ObjectBag, error) {
	var resources *resource.ObjectBag = new(resource.ObjectBag)
	r := rsrc.(*AirflowBase)
	if !s.Operator {
		resources.Add(
			*s.secret(r, rsrclabels),
			*s.service(r, rsrclabels),
			*s.sts(r, rsrclabels),
			*s.podDisruption(r, rsrclabels),
		)
	}
	//if s.VolumeClaimTemplate != nil {
	//	rsrcInfos = append(rsrcInfos, ResourceInfo{LifecycleReferred, s.VolumeClaimTemplate, ""})
	//}
	return resources, nil
}

// Observables - return selectors
func (s *MySQLSpec) Observables(scheme *runtime.Scheme, rsrc interface{}, rsrclabels map[string]string, expected *resource.ObjectBag) []resource.Observable {
	oo := resource.ObservablesFromObjects(scheme, expected, rsrclabels)
	return oo
}

// differs returns true if the resource needs to be updated
func differs(expected metav1.Object, observed metav1.Object) bool {
	switch expected.(type) {
	case *corev1.ServiceAccount:
		// Dont update a SA
		return false
	case *corev1.Secret:
		// Dont update a secret
		return false
	case *corev1.Service:
		expected.SetResourceVersion(observed.GetResourceVersion())
		expected.(*corev1.Service).Spec.ClusterIP = observed.(*corev1.Service).Spec.ClusterIP
	case *policyv1.PodDisruptionBudget:
		expected.SetResourceVersion(observed.GetResourceVersion())
	}
	return true
}

// Differs returns true if the resource needs to be updated
func (s *MySQLSpec) Differs(expected metav1.Object, observed metav1.Object) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *MySQLSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled []metav1.Object, err error) {
	if s != nil {
		stts := statusi.(*AirflowBaseStatus)
		stts.UpdateStatus(reconciled, err)
	}
}

// ------------------------------ POSTGRES  ---------------------------------------

func (s *PostgresSpec) service(r *AirflowBase, labels map[string]string) *resource.Object {
	return service(r, ValueAirflowComponentPostgres,
		rsrcName(r.Name, ValueAirflowComponentSQL, ""),
		labels,
		[]corev1.ServicePort{{Name: "postgres", Port: 5432}})
}

func (s *PostgresSpec) podDisruption(r *AirflowBase, labels map[string]string) *resource.Object {
	return podDisruption(r, ValueAirflowComponentPostgres, "", "100%", labels)
}

func (s *PostgresSpec) secret(r *AirflowBase, labels map[string]string) *resource.Object {
	name := rsrcName(r.Name, ValueAirflowComponentSQL, "")
	return &resource.Object{
		Obj: &corev1.Secret{
			ObjectMeta: r.getMeta(name, labels),
			Data: map[string][]byte{
				"password":     RandomAlphanumericString(16),
				"rootpassword": RandomAlphanumericString(16),
			},
		},
	}
}

func (s *PostgresSpec) sts(r *AirflowBase, labels map[string]string) *resource.Object {
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentSQL, "")
	ss := sts(r, ValueAirflowComponentPostgres, "", true, labels)
	ss.Spec.Replicas = &s.Replicas
	volName := "postgres-data"
	if s.VolumeClaimTemplate != nil {
		ss.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*s.VolumeClaimTemplate}
		volName = s.VolumeClaimTemplate.Name
	} else {
		ss.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
	}
	ss.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:  "postgres",
			Image: s.Image + ":" + s.Version,
			Env: []corev1.EnvVar{
				{Name: "POSTGRES_DB", Value: "testdb"},
				{Name: "POSTGRES_USER", Value: "airflow"},
				{Name: "POSTGRES_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "rootpassword")},
			},
			Resources: s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "postgres",
					ContainerPort: 5432,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/var/lib/postgres/data",
				},
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"bash", "-c", "psql -w -U $POSTGRES_USER -d $POSTGRES_DB -c SELECT 1"},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       20,
				TimeoutSeconds:      5,
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"bash", "-c", "psql -w -U $POSTGRES_USER -d $POSTGRES_DB -c SELECT 1"},
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       5,
				TimeoutSeconds:      2,
			},
		},
	}
	return &resource.Object{
		Obj:       ss,
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &appsv1.StatefulSetList{},
	}
}

// Mutate - mutate expected
func (s *PostgresSpec) Mutate(rsrc interface{}, status interface{}, expected, observed *resource.ObjectBag) (*resource.ObjectBag, error) {
	return expected, nil
}

// Finalize - execute finalizers
func (s *PostgresSpec) Finalize(rsrc, sts interface{}, observed *resource.ObjectBag) error {
	r := rsrc.(*AirflowBase)
	finalizer.Remove(r, finalizer.Cleanup)
	return nil
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *PostgresSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string) (*resource.ObjectBag, error) {
	var resources *resource.ObjectBag = new(resource.ObjectBag)
	r := rsrc.(*AirflowBase)
	if !s.Operator {
		resources.Add(
			*s.secret(r, rsrclabels),
			*s.service(r, rsrclabels),
			*s.sts(r, rsrclabels),
			*s.podDisruption(r, rsrclabels),
		)
	}
	//if s.VolumeClaimTemplate != nil {
	//	rsrcInfos = append(rsrcInfos, ResourceInfo{LifecycleReferred, s.VolumeClaimTemplate, ""})
	//}
	return resources, nil
}

// Observables - return selectors
func (s *PostgresSpec) Observables(scheme *runtime.Scheme, rsrc interface{}, rsrclabels map[string]string, expected *resource.ObjectBag) []resource.Observable {
	return resource.ObservablesFromObjects(scheme, expected, rsrclabels)
}

// Differs returns true if the resource needs to be updated
func (s *PostgresSpec) Differs(expected metav1.Object, observed metav1.Object) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *PostgresSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled []metav1.Object, err error) {
	if s != nil {
		stts := statusi.(*AirflowBaseStatus)
		stts.UpdateStatus(reconciled, err)
	}
}

// ------------------------------ Airflow UI ---------------------------------------

func (s *AirflowUISpec) secret(r *AirflowCluster, labels map[string]string) *resource.Object {
	name := rsrcName(r.Name, ValueAirflowComponentUI, "")
	return &resource.Object{
		Obj: &corev1.Secret{
			ObjectMeta: r.getMeta(name, labels),
			Data: map[string][]byte{
				"password": RandomAlphanumericString(16),
			},
		},
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &corev1.SecretList{},
	}
}

// Mutate - mutate expected
func (s *AirflowUISpec) Mutate(rsrc interface{}, status interface{}, expected, observed *resource.ObjectBag) (*resource.ObjectBag, error) {
	return expected, nil
}

// Finalize - execute finalizers
func (s *AirflowUISpec) Finalize(rsrc, sts interface{}, observed *resource.ObjectBag) error {
	r := rsrc.(*AirflowBase)
	finalizer.Remove(r, finalizer.Cleanup)
	return nil
}

// ExpectedResources returns the list of resource/name for those resources created by
func (s *AirflowUISpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string) (*resource.ObjectBag, error) {
	var resources *resource.ObjectBag = new(resource.ObjectBag)
	r := rsrc.(*AirflowCluster)
	resources.Add(*s.sts(r, rsrclabels), *s.secret(r, rsrclabels))
	return resources, nil
}

// Observables - return selectors
func (s *AirflowUISpec) Observables(scheme *runtime.Scheme, rsrc interface{}, rsrclabels map[string]string, expected *resource.ObjectBag) []resource.Observable {
	return resource.ObservablesFromObjects(scheme, expected, rsrclabels)
}

// Differs returns true if the resource needs to be updated
func (s *AirflowUISpec) Differs(expected metav1.Object, observed metav1.Object) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *AirflowUISpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled []metav1.Object, err error) {
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		stts.UpdateStatus(reconciled, err)
	}
}

func (s *AirflowUISpec) sts(r *AirflowCluster, labels map[string]string) *resource.Object {
	volName := "dags-data"
	ss := sts(r, ValueAirflowComponentUI, "", false, labels)
	ss.Spec.Replicas = &s.Replicas
	ss.Spec.PodManagementPolicy = PodManagementPolicyParallel
	ss.Spec.Template.Spec.Volumes = []corev1.Volume{
		{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	args := []string{"webserver"}
	env := r.getAirflowEnv(ss.Name)
	containers := []corev1.Container{
		{
			//imagePullPolicy: "Always"
			//envFrom:
			Name:            "airflow-ui",
			Image:           s.Image + ":" + s.Version,
			Env:             env,
			ImagePullPolicy: corev1.PullAlways,
			Args:            args,
			Resources:       s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "web",
					ContainerPort: 8080,
					//Protocol:      "TCP",
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/usr/local/airflow/dags/",
				},
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/health",
						Port: intstr.FromString("web"),
					},
				},
				InitialDelaySeconds: 100,
				PeriodSeconds:       60,
				TimeoutSeconds:      2,
				SuccessThreshold:    1,
				FailureThreshold:    5,
			},
		},
	}

	r.addAirflowContainers(ss, containers, volName)
	r.addMySQLUserDBContainer(ss)
	//	r.addPostgresUserDBContainer(ss)
	return &resource.Object{
		Obj:       ss,
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &appsv1.StatefulSetList{},
	}
}

// ------------------------------ NFSStoreSpec ---------------------------------------

// Mutate - mutate expected
func (s *NFSStoreSpec) Mutate(rsrc interface{}, status interface{}, expected, observed *resource.ObjectBag) (*resource.ObjectBag, error) {
	return expected, nil
}

// Finalize - execute finalizers
func (s *NFSStoreSpec) Finalize(rsrc, sts interface{}, observed *resource.ObjectBag) error {
	r := rsrc.(*AirflowBase)
	finalizer.Remove(r, finalizer.Cleanup)
	return nil
}

// ExpectedResources returns the list of resource/name for those resources created by
func (s *NFSStoreSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string) (*resource.ObjectBag, error) {
	var resources *resource.ObjectBag = new(resource.ObjectBag)
	r := rsrc.(*AirflowBase)
	resources.Add(
		*s.sts(r, rsrclabels),
		*s.service(r, rsrclabels),
		*s.podDisruption(r, rsrclabels),
	)
	return resources, nil
}

// Observables - return selectors
func (s *NFSStoreSpec) Observables(scheme *runtime.Scheme, rsrc interface{}, rsrclabels map[string]string, expected *resource.ObjectBag) []resource.Observable {
	return resource.ObservablesFromObjects(scheme, expected, rsrclabels)
}

func (s *NFSStoreSpec) sts(r *AirflowBase, labels map[string]string) *resource.Object {
	ss := sts(r, ValueAirflowComponentNFS, "", true, labels)
	ss.Spec.PodManagementPolicy = PodManagementPolicyParallel
	volName := "nfs-data"
	if s.Volume != nil {
		ss.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*s.Volume}
		volName = s.Volume.Name
	} else {
		ss.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
	}
	ss.Spec.Template.Spec.Containers = []corev1.Container{
		{
			//imagePullPolicy: "Always"
			//envFrom:
			Name:      "nfs-server",
			Image:     s.Image + ":" + s.Version,
			Resources: s.Resources,
			Ports: []corev1.ContainerPort{
				{Name: "nfs", ContainerPort: 2049},
				{Name: "mountd", ContainerPort: 20048},
				{Name: "rpcbind", ContainerPort: 111},
			},
			SecurityContext: &corev1.SecurityContext{},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/exports",
				},
			},
		},
	}
	return &resource.Object{
		Obj:       ss,
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &appsv1.StatefulSetList{},
	}
}

func (s *NFSStoreSpec) podDisruption(r *AirflowBase, labels map[string]string) *resource.Object {
	return podDisruption(r, ValueAirflowComponentNFS, "", "100%", labels)
}

func (s *NFSStoreSpec) service(r *AirflowBase, labels map[string]string) *resource.Object {
	return service(r, ValueAirflowComponentNFS, "", labels,
		[]corev1.ServicePort{
			{Name: "nfs", Port: 2049},
			{Name: "mountd", Port: 20048},
			{Name: "rpcbind", Port: 111},
		})
}

// Differs returns true if the resource needs to be updated
func (s *NFSStoreSpec) Differs(expected metav1.Object, observed metav1.Object) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *NFSStoreSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled []metav1.Object, err error) {
	if s != nil {
		stts := statusi.(*AirflowBaseStatus)
		stts.UpdateStatus(reconciled, err)
	}
}

// ------------------------------ SQLProxy ---------------------------------------
func (s *SQLProxySpec) service(r *AirflowBase, labels map[string]string) *resource.Object {
	return service(r, ValueAirflowComponentSQLProxy,
		rsrcName(r.Name, ValueAirflowComponentSQL, ""), labels,
		[]corev1.ServicePort{{Name: "sqlproxy", Port: 3306}})
}

func (s *SQLProxySpec) sts(r *AirflowBase, labels map[string]string) *resource.Object {
	ss := sts(r, ValueAirflowComponentSQLProxy, "", true, labels)
	instance := s.Project + ":" + s.Region + ":" + s.Instance + "=tcp:0.0.0.0:3306"
	ss.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:  "sqlproxy",
			Image: s.Image + ":" + s.Version,
			Env: []corev1.EnvVar{
				{Name: "SQL_INSTANCE", Value: instance},
			},
			Command: []string{"/cloud_sql_proxy", "-instances", "$(SQL_INSTANCE)"},
			//volumeMounts:
			//- name: ssl-certs
			//mountPath: /etc/ssl/certs
			Resources: s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "sqlproxy",
					ContainerPort: 3306,
				},
			},
		},
	}
	return &resource.Object{
		Obj:       ss,
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &appsv1.StatefulSetList{},
	}
}

// Mutate - mutate expected
func (s *SQLProxySpec) Mutate(rsrc interface{}, status interface{}, expected, observed *resource.ObjectBag) (*resource.ObjectBag, error) {
	return expected, nil
}

// Finalize - execute finalizers
func (s *SQLProxySpec) Finalize(rsrc, sts interface{}, observed *resource.ObjectBag) error {
	r := rsrc.(*AirflowBase)
	finalizer.Remove(r, finalizer.Cleanup)
	return nil
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *SQLProxySpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string) (*resource.ObjectBag, error) {
	var resources *resource.ObjectBag = new(resource.ObjectBag)
	r := rsrc.(*AirflowBase)
	name := rsrcName(r.Name, ValueAirflowComponentSQLProxy, "")
	resources.Add(
		*s.service(r, rsrclabels),
		*s.sts(r, rsrclabels),
		resource.Object{
			Lifecycle: resource.LifecycleReferred,
			Obj: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: r.Namespace,
					Name:      name,
				},
			},
		})
	return resources, nil
}

// Observables - return selectors
func (s *SQLProxySpec) Observables(scheme *runtime.Scheme, rsrc interface{}, rsrclabels map[string]string, expected *resource.ObjectBag) []resource.Observable {
	return resource.ObservablesFromObjects(scheme, expected, rsrclabels)
}

// Differs returns true if the resource needs to be updated
func (s *SQLProxySpec) Differs(expected metav1.Object, observed metav1.Object) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *SQLProxySpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled []metav1.Object, err error) {
	if s != nil {
		stts := statusi.(*AirflowBaseStatus)
		stts.UpdateStatus(reconciled, err)
	}
}

// ------------------------------ RedisSpec ---------------------------------------
func (s *RedisSpec) service(r *AirflowCluster, labels map[string]string) *resource.Object {
	return service(r, ValueAirflowComponentRedis, "", labels,
		[]corev1.ServicePort{{Name: "redis", Port: 6379}})
}

func (s *RedisSpec) podDisruption(r *AirflowCluster, labels map[string]string) *resource.Object {
	return podDisruption(r, ValueAirflowComponentRedis, "", "100%", labels)
}

func (s *RedisSpec) secret(r *AirflowCluster, labels map[string]string) *resource.Object {
	name := rsrcName(r.Name, ValueAirflowComponentRedis, "")
	return &resource.Object{
		ObjList:   &corev1.SecretList{},
		Lifecycle: resource.LifecycleManaged,
		Obj: &corev1.Secret{
			ObjectMeta: r.getMeta(name, labels),
			Data: map[string][]byte{
				"password": RandomAlphanumericString(16),
			},
		},
	}
}

func (s *RedisSpec) sts(r *AirflowCluster, labels map[string]string) *resource.Object {
	redisSecret := rsrcName(r.Name, ValueAirflowComponentRedis, "")
	ss := sts(r, ValueAirflowComponentRedis, "", true, labels)
	volName := "redis-data"
	if s.VolumeClaimTemplate != nil {
		ss.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*s.VolumeClaimTemplate}
		volName = s.VolumeClaimTemplate.Name
	} else {
		ss.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
	}
	args := []string{"--requirepass", "$(REDIS_PASSWORD)"}
	if s.AdditionalArgs != "" {
		args = append(args, "$(REDIS_EXTRA_FLAGS)")
	}
	ss.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:  "redis",
			Image: s.Image + ":" + s.Version,
			Env: []corev1.EnvVar{
				{Name: "REDIS_EXTRA_FLAGS", Value: s.AdditionalArgs},
				{Name: "REDIS_PASSWORD", ValueFrom: envFromSecret(redisSecret, "password")},
			},
			Args:      args,
			Resources: s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "redis",
					ContainerPort: 6379,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/data",
				},
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"redis-cli", "ping"},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       20,
				TimeoutSeconds:      5,
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"redis-cli", "ping"},
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       5,
				TimeoutSeconds:      2,
			},
		},
	}
	return &resource.Object{
		Obj:       ss,
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &appsv1.StatefulSetList{},
	}
}

// Mutate - mutate expected
func (s *RedisSpec) Mutate(rsrc interface{}, status interface{}, expected, observed *resource.ObjectBag) (*resource.ObjectBag, error) {
	return expected, nil
}

// Finalize - execute finalizers
func (s *RedisSpec) Finalize(rsrc, sts interface{}, observed *resource.ObjectBag) error {
	r := rsrc.(*AirflowBase)
	finalizer.Remove(r, finalizer.Cleanup)
	return nil
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *RedisSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string) (*resource.ObjectBag, error) {
	var resources *resource.ObjectBag = new(resource.ObjectBag)
	r := rsrc.(*AirflowCluster)
	resources.Add(
		*s.secret(r, rsrclabels),
		*s.service(r, rsrclabels),
		*s.sts(r, rsrclabels),
		*s.podDisruption(r, rsrclabels),
	)
	return resources, nil
	//if s.VolumeClaimTemplate != nil {
	//		rsrcInfos = append(rsrcInfos, ResourceInfo{LifecycleReferred, s.VolumeClaimTemplate, ""})
	//	}
}

// Observables - return selectors
func (s *RedisSpec) Observables(scheme *runtime.Scheme, rsrc interface{}, rsrclabels map[string]string, expected *resource.ObjectBag) []resource.Observable {
	return resource.ObservablesFromObjects(scheme, expected, rsrclabels)
}

// Differs returns true if the resource needs to be updated
func (s *RedisSpec) Differs(expected metav1.Object, observed metav1.Object) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *RedisSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled []metav1.Object, err error) {
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		stts.UpdateStatus(reconciled, err)
	}
}

// ------------------------------ Scheduler ---------------------------------------

func (s *GCSSpec) container(volName string) (bool, corev1.Container) {
	init := false
	container := corev1.Container{}
	env := []corev1.EnvVar{
		{Name: "GCS_BUCKET", Value: s.Bucket},
	}
	if s.Once {
		init = true
	}
	container = corev1.Container{
		Name:  "gcs-syncd",
		Image: gcssyncImage + ":" + gcssyncVersion,
		Env:   env,
		Args:  []string{"/home/airflow/gcs"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volName,
				MountPath: "/home/airflow/gcs",
			},
		},
	}

	return init, container
}
func (s *GitSpec) container(volName string) (bool, corev1.Container) {
	init := false
	container := corev1.Container{}
	env := []corev1.EnvVar{
		{Name: "GIT_SYNC_REPO", Value: s.Repo},
		{Name: "GIT_SYNC_DEST", Value: GitSyncDestDir},
		{Name: "GIT_SYNC_BRANCH", Value: s.Branch},
		{Name: "GIT_SYNC_ONE_TIME", Value: strconv.FormatBool(s.Once)},
		{Name: "GIT_SYNC_REV", Value: s.Rev},
	}
	if s.CredSecretRef != nil {
		env = append(env, []corev1.EnvVar{
			{Name: "GIT_PASSWORD",
				ValueFrom: envFromSecret(s.CredSecretRef.Name, "password")},
			{Name: "GIT_USER", Value: s.User},
		}...)
	}
	if s.Once {
		init = true
	}
	container = corev1.Container{
		Name:    "git-sync",
		Image:   gitsyncImage + ":" + gitsyncVersion,
		Env:     env,
		Command: []string{"/git-sync"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "gitsync",
				ContainerPort: 2020,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volName,
				MountPath: "/git",
			},
		},
	}

	return init, container
}

func (s *DagSpec) container(volName string) (bool, corev1.Container) {
	init := false
	container := corev1.Container{}

	if s.Git != nil {
		return s.Git.container(volName)
	}
	if s.GCS != nil {
		return s.GCS.container(volName)
	}

	return init, container
}

func (s *SchedulerSpec) serviceaccount(r *AirflowCluster, labels map[string]string) *resource.Object {
	name := rsrcName(r.Name, ValueAirflowComponentScheduler, "")
	return &resource.Object{
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &corev1.ServiceAccountList{},
		Obj: &corev1.ServiceAccount{
			ObjectMeta: r.getMeta(name, labels),
		},
	}
}

func (s *SchedulerSpec) rb(r *AirflowCluster, labels map[string]string) *resource.Object {
	name := rsrcName(r.Name, ValueAirflowComponentScheduler, "")
	return &resource.Object{
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &rbacv1.RoleBindingList{},
		Obj: &rbacv1.RoleBinding{
			ObjectMeta: r.getMeta(name, labels),
			Subjects: []rbacv1.Subject{
				{Kind: "ServiceAccount", Name: name, Namespace: r.Namespace},
			},
			RoleRef: rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "cluster-admin"},
		},
	}
}

func (s *SchedulerSpec) sts(r *AirflowCluster, labels map[string]string) *resource.Object {
	volName := "dags-data"
	ss := sts(r, ValueAirflowComponentScheduler, "", true, labels)
	ss.Spec.Template.Spec.Volumes = []corev1.Volume{
		{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	args := []string{"scheduler"}

	if r.Spec.Executor == ExecutorK8s {
		ss.Spec.Template.Spec.ServiceAccountName = ss.Name
	}
	containers := []corev1.Container{
		{
			Name:            "scheduler",
			Image:           s.Image + ":" + s.Version,
			Env:             r.getAirflowEnv(ss.Name),
			ImagePullPolicy: corev1.PullAlways,
			Args:            args,
			Resources:       s.Resources,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/usr/local/airflow/dags/",
				},
			},
		},
		{
			Name:  "metrics",
			Image: "pbweb/airflow-prometheus-exporter:latest",
			Env:   r.getAirflowPrometheusEnv(),
			Ports: []corev1.ContainerPort{
				{
					Name:          "metrics",
					ContainerPort: 9112,
				},
			},
		},
	}
	r.addAirflowContainers(ss, containers, volName)
	return &resource.Object{
		Obj:       ss,
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &appsv1.StatefulSetList{},
	}
}

// Mutate - mutate expected
func (s *SchedulerSpec) Mutate(rsrc interface{}, status interface{}, expected, observed *resource.ObjectBag) (*resource.ObjectBag, error) {
	return expected, nil
}

// Finalize - execute finalizers
func (s *SchedulerSpec) Finalize(rsrc, sts interface{}, observed *resource.ObjectBag) error {
	r := rsrc.(*AirflowBase)
	finalizer.Remove(r, finalizer.Cleanup)
	return nil
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *SchedulerSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string) (*resource.ObjectBag, error) {
	var resources *resource.ObjectBag = new(resource.ObjectBag)
	r := rsrc.(*AirflowCluster)
	resources.Add(
		*s.serviceaccount(r, rsrclabels),
		*s.rb(r, rsrclabels),
		*s.sts(r, rsrclabels),
	)

	if r.Spec.DAGs != nil {
		git := r.Spec.DAGs.Git
		if git != nil && git.CredSecretRef != nil {
			resources.Add(resource.Object{
				Lifecycle: resource.LifecycleReferred,
				Obj: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: r.Namespace,
						Name:      git.CredSecretRef.Name,
					},
				},
			})
		}
	}

	return resources, nil
}

// Observables - return selectors
func (s *SchedulerSpec) Observables(scheme *runtime.Scheme, rsrc interface{}, rsrclabels map[string]string, expected *resource.ObjectBag) []resource.Observable {
	return resource.ObservablesFromObjects(scheme, expected, rsrclabels)
}

// Differs returns true if the resource needs to be updated
func (s *SchedulerSpec) Differs(expected metav1.Object, observed metav1.Object) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *SchedulerSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled []metav1.Object, err error) {
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		stts.UpdateStatus(reconciled, err)
	}
}

// ------------------------------ Worker -
func (s *WorkerSpec) sts(r *AirflowCluster, labels map[string]string) *resource.Object {
	ss := sts(r, ValueAirflowComponentWorker, "", true, labels)
	volName := "dags-data"
	ss.Spec.Replicas = &s.Replicas
	ss.Spec.Template.Spec.Volumes = []corev1.Volume{
		{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	args := []string{"worker"}
	env := r.getAirflowEnv(ss.Name)
	containers := []corev1.Container{
		{
			Name:            "worker",
			Image:           s.Image + ":" + s.Version,
			Args:            args,
			Env:             env,
			ImagePullPolicy: corev1.PullAlways,
			Resources:       s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "wlog",
					ContainerPort: 8793,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/usr/local/airflow/dags/",
				},
			},
		},
	}
	r.addAirflowContainers(ss, containers, volName)
	return &resource.Object{
		Obj:       ss,
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &appsv1.StatefulSetList{},
	}
}

// Mutate - mutate expected
func (s *WorkerSpec) Mutate(rsrc interface{}, status interface{}, expected, observed *resource.ObjectBag) (*resource.ObjectBag, error) {
	return expected, nil
}

// Finalize - execute finalizers
func (s *WorkerSpec) Finalize(rsrc, sts interface{}, observed *resource.ObjectBag) error {
	r := rsrc.(*AirflowBase)
	finalizer.Remove(r, finalizer.Cleanup)
	return nil
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *WorkerSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string) (*resource.ObjectBag, error) {
	var resources *resource.ObjectBag = new(resource.ObjectBag)
	r := rsrc.(*AirflowCluster)
	resources.Add(*s.sts(r, rsrclabels))
	return resources, nil
}

// Observables - return selectors
func (s *WorkerSpec) Observables(scheme *runtime.Scheme, rsrc interface{}, rsrclabels map[string]string, expected *resource.ObjectBag) []resource.Observable {
	return resource.ObservablesFromObjects(scheme, expected, rsrclabels)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *WorkerSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled []metav1.Object, err error) {
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		stts.UpdateStatus(reconciled, err)
	}
}

// Differs returns true if the resource needs to be updated
func (s *WorkerSpec) Differs(expected metav1.Object, observed metav1.Object) bool {
	// TODO
	return true
}

// ------------------------------ Flower ---------------------------------------

// Mutate - mutate expected
func (s *FlowerSpec) Mutate(rsrc interface{}, status interface{}, expected, observed *resource.ObjectBag) (*resource.ObjectBag, error) {
	return expected, nil
}

// Finalize - execute finalizers
func (s *FlowerSpec) Finalize(rsrc, sts interface{}, observed *resource.ObjectBag) error {
	r := rsrc.(*AirflowBase)
	finalizer.Remove(r, finalizer.Cleanup)
	return nil
}

// ExpectedResources returns the list of resource/name for those resources created by
func (s *FlowerSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string) (*resource.ObjectBag, error) {
	var resources *resource.ObjectBag = new(resource.ObjectBag)
	r := rsrc.(*AirflowCluster)
	resources.Add(*s.sts(r, rsrclabels))
	return resources, nil
}

// Observables - return selectors
func (s *FlowerSpec) Observables(scheme *runtime.Scheme, rsrc interface{}, rsrclabels map[string]string, expected *resource.ObjectBag) []resource.Observable {
	return resource.ObservablesFromObjects(scheme, expected, rsrclabels)
}

// Differs returns true if the resource needs to be updated
func (s *FlowerSpec) Differs(expected metav1.Object, observed metav1.Object) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *FlowerSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled []metav1.Object, err error) {
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		stts.UpdateStatus(reconciled, err)
	}
}

func (s *FlowerSpec) sts(r *AirflowCluster, labels map[string]string) *resource.Object {
	ss := sts(r, ValueAirflowComponentFlower, "", true, labels)
	volName := "dags-data"
	ss.Spec.Replicas = &s.Replicas
	ss.Spec.Template.Spec.Volumes = []corev1.Volume{
		{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	args := []string{"flower"}
	env := r.getAirflowEnv(ss.Name)
	containers := []corev1.Container{
		{
			Name:            "flower",
			Image:           s.Image + ":" + s.Version,
			Args:            args,
			Env:             env,
			ImagePullPolicy: corev1.PullAlways,
			Resources:       s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "flower",
					ContainerPort: 5555,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/usr/local/airflow/dags/",
				},
			},
		},
	}
	r.addAirflowContainers(ss, containers, volName)
	return &resource.Object{
		Obj:       ss,
		Lifecycle: resource.LifecycleManaged,
		ObjList:   &appsv1.StatefulSetList{},
	}
}