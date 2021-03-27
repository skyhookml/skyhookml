import utils from './utils.js';
import JobConsoleProgress from './job-consoleprogress.js';
import JobPytorchTrain from './job-pytorch_train.js';

export default {
	components: {
		'job-consoleprogress': JobConsoleProgress,
		'job-pytorch_train': JobPytorchTrain,
	},
	data: function() {
		return {
			job: null,
			plan: [],
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
			utils.request(this, 'POST', '/jobs/'+this.jobID+'/state', null, (state) => {
				if(!state) {
					return;
				}
				this.job = state.CurJob;
				this.plan = state.Plan;
			});
		},
	},
	template: `
<div class="el-high">
	<component v-if="job" v-bind:is="'job-'+job.Op" v-bind:jobID="job.ID"></component>
</div>
	`,
};
