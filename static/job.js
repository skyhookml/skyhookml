import utils from './utils.js';
import JobConsoleProgress from './job-consoleprogress.js';
import JobPytorchTrain from './job-pytorch_train.js';
import JobMultiExec from './job-multi_exec.js';

export default {
	components: {
		'job-consoleprogress': JobConsoleProgress,
		'job-pytorch_train': JobPytorchTrain,
		'job-multiexec': JobMultiExec,
	},
	data: function() {
		return {
			job: null,
		};
	},
	created: function() {
		utils.request(this, 'GET', '/jobs/'+this.$route.params.jobid, null, (job) => {
			this.job = job;
		});
	},
	template: `
<div class="el-high">
	<component v-if="job" v-bind:is="'job-'+job.Op" v-bind:jobID="job.ID"></component>
</div>
	`,
};