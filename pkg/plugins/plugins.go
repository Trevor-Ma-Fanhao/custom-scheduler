package plugins

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Trevor-Ma-Fanhao/custom-scheduler/pkg/forest"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	"k8s.io/kubernetes/pkg/scheduler/nodeinfo"
)

const (
	// 插件名称
	Name = "custom-plugin"
	// PredictedResourcesKey is the key for the custom state in CycleState
	PredictedResourcesKey = "PredictedResources"
)

type PredictedResources struct {
	gpuCores  int
	gpumem    int
	bandwidth int
}

// Clone returns a deep copy of PredictedResources.
func (pr *PredictedResources) Clone() framework.StateData {
	return &PredictedResources{
		gpuCores:  pr.gpuCores,
		gpumem:    pr.gpumem,
		bandwidth: pr.bandwidth,
	}
}

type Args struct {
	FavoriteColor  string `json:"favorite_color,omitempty"`
	FavoriteNumber int    `json:"favorite_number,omitempty"`
	ThanksTo       string `json:"thanks_to,omitempty"`
}

type Sample struct {
	args   *Args
	handle framework.FrameworkHandle
	model1 *forest.Regressor
	model2 *forest.Regressor
	model3 *forest.Regressor
}

func (s *Sample) Name() string {
	return Name
}

// PreFilterExtensions 返回调度器调用 AddPod/RemovePod 方法以评估对节点的改变的影响时的插件接口。
func (s *Sample) PreFilterExtensions() framework.PreFilterExtensions {
	fmt.Println("进入PreFilterExtensions........")
	fmt.Println("-----------------------------------------")
	return s
}

func (s *Sample) AddPod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podToAdd *v1.Pod, nodeInfo *nodeinfo.NodeInfo) *framework.Status {
	fmt.Println("进入AddPod...............")
	fmt.Println("-----------------------------------------")
	return framework.NewStatus(framework.Success)
}

func (s *Sample) RemovePod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podToRemove *v1.Pod, nodeInfo *nodeinfo.NodeInfo) *framework.Status {
	fmt.Println("进入RemovePod...............")
	fmt.Println("-----------------------------------------")
	return framework.NewStatus(framework.Success)
}

// 转换注解中字符串为整型
func parseAnnotation(pod *v1.Pod, key string) (int, error) {
	if value, ok := pod.Annotations[key]; ok {
		return strconv.Atoi(value)
	}
	return 0, fmt.Errorf("annotation %s not found", key)
}

func (sr *Sample) predictResources(cycleState *framework.CycleState, pod *v1.Pod) error {
	// 从 Pod 的注解中提取 featureData
	batchSize, _ := parseAnnotation(pod, "batch_size")
	numBatches, _ := parseAnnotation(pod, "num_batches")
	numPS, _ := parseAnnotation(pod, "num_ps")
	numWorker, _ := parseAnnotation(pod, "num_worker")
	fmt.Printf("Feature input : batchSize: %d, numBatches: %d, numPS: %d, numWorker: %d\n", batchSize, numBatches, numPS, numWorker)
	// 用于模型预测的数据结构（一个 slice of floats）
	featureData := []float64{float64(batchSize), float64(numBatches), float64(numPS), float64(numWorker)}
	featureData2 := [][]float64{featureData}
	// 使用模型进行预测
	pred1 := sr.model1.Predict(featureData2)
	pred2 := sr.model2.Predict(featureData2)
	pred3 := sr.model3.Predict(featureData2)
	predictedResources := PredictedResources{
		gpuCores:  int(pred1[0]),
		gpumem:    int(pred2[0]),
		bandwidth: int(pred3[0]),
	}
	// 打印预测结果
	fmt.Printf("Predicted resources - GPU Cores: %d, GPU Memory: %d, Bandwidth: %d\n",
		predictedResources.gpuCores,
		predictedResources.gpumem,
		predictedResources.bandwidth,
	)
	// Store the predicted resources in CycleState
	cycleState.Write(PredictedResourcesKey, &predictedResources)

	return nil
}

// This is an example within a Filter or Score plugin method
func getPredictedResources(cycleState *framework.CycleState) (*PredictedResources, error) {
	c, err := cycleState.Read(PredictedResourcesKey)
	if err != nil {
		// no prediction data available, handle error appropriately
		return nil, fmt.Errorf("error reading %s from cycleState: %v", PredictedResourcesKey, err)
	}

	predictedResources, ok := c.(*PredictedResources)
	if !ok {
		// the stored data is not of the expected type
		return nil, fmt.Errorf("expected *PredictedResources type, but got %T", c)
	}

	return predictedResources, nil
}

func (sr Sample) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) *framework.Status {
	fmt.Println("进入PreFilter........")
	klog.V(3).Infof("prefilter pod: %v", pod.Name)
	// 调用 predictResources
	if err := sr.predictResources(state, pod); err != nil {
		// 如果有错误发生，应返回状态并记录错误
		klog.Errorf("Error predicting resources for pod %s: %v", pod.Name, err)
		return framework.NewStatus(framework.Error, err.Error())
	}
	// 访问 CycleState 以获取预测资源并打印
	//v, err := state.Read(PredictedResourcesKey)
	//if err != nil {
	//    klog.Errorf("Error reading PredictedResourcesKey from CycleState: %v", err)
	//    return framework.NewStatus(framework.Error, err.Error())
	//}

	//predictedResources, ok := v.(*PredictedResources)
	//if !ok {
	// 无法断言正确的类型，记录错误
	//    errorMessage := fmt.Sprintf("Want *PredictedResources type, got %T type", v)
	//    klog.Error(errorMessage)
	//    return framework.NewStatus(framework.Error, errorMessage)
	//}
	predictedResources, err := getPredictedResources(state)
	if err != nil {
		fmt.Println(err)
	} else {
		// 打印预测资源
		fmt.Printf("Prefilter中的 Predicted resources - GPU Cores: %d, GPU Memory: %d, Bandwidth: %d\n",
			predictedResources.gpuCores,
			predictedResources.gpumem,
			predictedResources.bandwidth,
		)
	}
	return framework.NewStatus(framework.Success, "")
}

func (sr Sample) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeinfo *nodeinfo.NodeInfo) *framework.Status {
	fmt.Println("进入Filter........")
	nodeName := nodeinfo.Node().Name
	klog.V(3).Infof("filter pod: %v, node: %v", pod.Name, nodeName)
	return framework.NewStatus(framework.Success, "")
}

func (sr Sample) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	fmt.Println("进入Score........")
	fmt.Printf("pod:%s, namespace:%s\n", pod.Name, pod.Namespace)
	fmt.Printf("nodeName:%s\n", nodeName)
	fmt.Println("-----------------------------------------")
	// 获取节点的信息
	nodeInfo, err := sr.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		// 处理错误，例如打印日志、返回错误等
		fmt.Println("获取节点信息失败:", err)
		return 100, framework.NewStatus(framework.Success)
	} else {
		fmt.Println("获取节点信息成功:")
	}
	fmt.Println(nodeInfo.String())
	resource := nodeInfo.AllocatableResource() // 调用函数获取 nodeinfo.Resource 实例
	milliCPU := resource.MilliCPU              // 访问 MilliCPU 字段
	memory := resource.Memory                  // 访问 Memory 字段
	fmt.Printf("MilliCPU: %d, 类型: %T\n", milliCPU, milliCPU)
	fmt.Printf("Memory: %d, 类型: %T\n", memory, memory)

	predictedResources, err := getPredictedResources(state)
	cpuScore := milliCPU / int64(predictedResources.gpuCores)
	memoryScore := memory / (int64(predictedResources.gpumem) * 1024)
	score := (cpuScore + memoryScore) / 2
	fmt.Println("score of node %s: %d", nodeName, score)
	return score, framework.NewStatus(framework.Success)
}

func (s *Sample) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	fmt.Println("进入NormalizeScore........")
	fmt.Printf("pod:%s,namespace:%s\n", pod.Name, pod.Namespace)
	fmt.Println("-----------------------------------------")
	var maxScore int64 = 0
	for _, ns := range scores {
		if ns.Score > maxScore {
			maxScore = ns.Score
		}
	}
	if maxScore == 0 {
		return framework.NewStatus(framework.Success)
	}
	// 进行归一化处理
	for i := range scores {
		scores[i].Score = scores[i].Score * framework.MaxNodeScore / maxScore
		fmt.Printf("the %v of scores is %v\n", i, scores[i].Score)
	}
	return framework.NewStatus(framework.Success)
}

// 注意更新 ScoreExtensions 方法来返回包含 NormalizeScore 方法实现的实例。
func (sr *Sample) ScoreExtensions() framework.ScoreExtensions {
	fmt.Println("进入ScoreExtensions........")
	fmt.Println("-----------------------------------------")
	return sr
}

func (sr Sample) PreBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	fmt.Println("进入PreBind........")
	fmt.Printf("pod:%s,namespace:%s\n", pod.Name, pod.Namespace)
	fmt.Printf("nodeName:%v\n", nodeName)
	fmt.Println("-----------------------------------------")

	// if pod == nil {
	//         return framework.NewStatus(framework.Error, fmt.Sprintf("pod cannot be nil"))
	// }
	// if !strings.Contains(pod.Name, "mypod") {
	//         return framework.NewStatus(framework.Error, fmt.Sprintf("pod name need contain mypod"))
	//         //binding := &v1.Binding{
	//         //      ObjectMeta: v12.ObjectMeta{Namespace: pod.Namespace, Name: pod.Name, UID: pod.UID},
	//         //      Target:     v1.ObjectReference{Kind: "Node", Name: "master1"},
	//         //}
	//         //sr.handle.ClientSet().CoreV1().Pods(pod.Namespace).Bind(ctx,binding,v12.CreateOptions{})
	// }
	return framework.NewStatus(framework.Success, "")
}

func (sr Sample) Bind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	fmt.Println("进入Bind........")
	fmt.Printf("pod:%s,namespace:%s\n", pod.Name, pod.Namespace)
	fmt.Printf("nodeName:%v\n", nodeName)
	fmt.Println("-----------------------------------------")
	if pod == nil {
		return framework.NewStatus(framework.Error, fmt.Sprintf("pod cannot be nil"))
	}
	binding := &v1.Binding{
		ObjectMeta: v12.ObjectMeta{Namespace: pod.Namespace, Name: pod.Name, UID: pod.UID},
		Target:     v1.ObjectReference{Kind: "Node", Name: nodeName},
	}
	err := sr.handle.ClientSet().CoreV1().Pods(pod.Namespace).Bind(ctx, binding, v12.CreateOptions{})
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error())
	}
	return nil
}

// type PluginFactory = func(configuration *runtime.Unknown, f FrameworkHandle) (Plugin, error)
func New(configuration *runtime.Unknown, f framework.FrameworkHandle) (framework.Plugin, error) {
	args := &Args{}
	if err := framework.DecodeInto(configuration, args); err != nil {
		return nil, err
	}
	klog.V(3).Infof("get plugin config args: %+v", args)

	// 创建随机森林模型
	reg1 := forest.NewRegressor(
		forest.NumTrees(10),
		forest.MaxFeatures(3),
		forest.MinSplit(2),
		forest.MinLeaf(1),
		forest.MaxDepth(10),
		forest.NumWorkers(1),
	)
	// 创建随机森林模型
	reg2 := forest.NewRegressor(
		forest.NumTrees(10),
		forest.MaxFeatures(3),
		forest.MinSplit(2),
		forest.MinLeaf(1),
		forest.MaxDepth(10),
		forest.NumWorkers(1),
	)
	// 创建随机森林模型
	reg3 := forest.NewRegressor(
		forest.NumTrees(10),
		forest.MaxFeatures(3),
		forest.MinSplit(2),
		forest.MinLeaf(1),
		forest.MaxDepth(10),
		forest.NumWorkers(1),
	)
	// 使用训练数据进行训练
	reg1.Fit(featureData, Num_GPU)
	reg2.Fit(featureData, GMem)
	reg3.Fit(featureData, Bandwidth)

	return &Sample{
		args:   args,
		handle: f,
		model1: reg1,
		model2: reg2,
		model3: reg3,
	}, nil
}

var Features = []string{"batch_size", "num_batches", "num_ps", "num_worker"}

var featureData = [][]float64{
	[]float64{32, 100, 1, 4},
	[]float64{16, 50, 2, 8},
	[]float64{64, 200, 4, 16},
	[]float64{128, 500, 8, 32},
	[]float64{256, 1000, 16, 64},
}

var Num_GPU = []float64{
	1.0,  // 对应可能的 GPU 个数
	2.0,  // 对应可能的 GPU 个数
	4.0,  // 对应可能的 GPU 个数
	8.0,  // 对应可能的 GPU 个数
	16.0, // 对应可能的 GPU 个数
}

var GMem = []float64{
	4.0,  // 对应可能的 GPU 内存大小，以 GB 为单位
	8.0,  // 对应可能的 GPU 内存大小，以 GB 为单位
	16.0, // 对应可能的 GPU 内存大小，以 GB 为单位
	32.0, // 对应可能的 GPU 内存大小，以 GB 为单位
	64.0, // 对应可能的 GPU 内存大小，以 GB 为单位
}

var Bandwidth = []float64{
	80.0,  // 对应可能的带宽大小，以 GB/s 为单位
	120.0, // 对应可能的带宽大小，以 GB/s 为单位
	160.0, // 对应可能的带宽大小，以 GB/s 为单位
	240.0, // 对应可能的带宽大小，以 GB/s 为单位
	320.0, // 对应可能的带宽大小，以 GB/s 为单位
}
