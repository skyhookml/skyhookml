<template>
<div class="flex-container">
	<div class="chartjs-container el-50h">
		<canvas ref="layer"></canvas>
	</div>
	<div class="el-50h flex-container">
		<job-console :lines="lines"></job-console>
		<div v-if="job && !job.Done" class="mb-2">
			<button class="btn btn-warning" v-on:click="stopTraining" data-bs-toggle="tooltip" title="Terminate the job, and mark the currently saved model as completed.">Stop Training and Mark Done</button>
		</div>
		<job-footer :job="job"></job-footer>
	</div>
</div>
</template>

<script>
import utils from './utils.js';
import JobConsole from './job-console.vue';
import JobFooter from './job-footer.vue';

export default {
	components: {
		'job-console': JobConsole,
		'job-footer': JobFooter,
	},
	data: function() {
		return {
			job: null,
			modelState: null,
			lines: [],
			chart: null,
		};
	},
	props: ['jobID'],
	created: function() {
		this.fetch();
		this.interval = setInterval(this.fetch, 1000);
	},
	destroyed: function() {
		clearInterval(this.interval);
	},
	methods: {
		fetch: function() {
			utils.request(this, 'POST', '/jobs/'+this.jobID+'/state', null, (response) => {
				this.job = response.Job;
				let state;
				try {
					state = JSON.parse(response.State);
				} catch(e) {}
				if(!state) {
					return;
				}
				let metadata = null;
				try {
					metadata = JSON.parse(state.Datas.node);
				} catch(e) {}
				this.updateChart(metadata);
				this.lines = state.Lines;
			});
		},
		updateChart: function(modelState) {
			if(!modelState || !modelState.TrainLoss || modelState.TrainLoss.length == 0) {
				return;
			}
			let n = modelState.TrainLoss.length;
			let prevN = 0;
			if(this.modelState) {
				prevN = this.modelState.TrainLoss.length;
			}
			if(prevN == n) {
				return;
			}
			if(!this.chart) {
				let labels = [];
				for(let i = 0; i < n; i++) {
					labels.push('Epoch ' + i);
				}
				let config = {
					type: 'line',
					data: {
						labels: labels,
						datasets: [{
							label: 'Train Loss',
							data: modelState.TrainLoss,
							fill: false,
							backgroundColor: 'blue',
							borderColor: 'blue',
						}, {
							label: 'Validation Loss',
							data: modelState.ValLoss,
							fill: false,
							backgroundColor: 'red',
							borderColor: 'red',
						}]
					},
					options: {
						responsive: true,
						maintainAspectRatio: false,
					},
				};
				let ctx = this.$refs.layer.getContext('2d');
				this.chart = new Chart(ctx, config);
			} else {
				// update chart with only the new history
				for(let i = prevN; i < n; i++) {
					this.chart.data.labels.push('Epoch ' + i);
					this.chart.data.datasets[0].data.push(modelState.TrainLoss[i]);
					this.chart.data.datasets[1].data.push(modelState.ValLoss[i]);
				}
				this.chart.update();
			}
			this.modelState = modelState;
		},
		stopTraining: function() {
			// We provide functionality to terminate the job while marking the dataset
			// done so that the user doesn't have to wait a long time for training to
			// complete if they are satisfied with the model performance.
			// This is NOT recommended since it breaks the (non-deterministic)
			// reproducibility of pipelines, but it seems important for users to have
			// this option.
			const jobID = this.job.ID;
			const nodeID = this.job.Metadata;
			// run an anonymous async function
			(async () => {
				console.log('[stop-training]', 'stopping job ' + this.job.Name);
				await utils.request(this, 'POST', '/jobs/'+jobID+'/stop');
				console.log('[stop-training]', 'marking outputs of node ' + nodeID + ' as done');
				await utils.request(this, 'POST', '/exec-nodes/'+nodeID+'/set-done');
			})();
		},
	},
};
</script>
