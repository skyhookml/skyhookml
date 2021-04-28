<template>
<div class="el-high">
	<component v-if="job" v-bind:is="'job-'+job.Op" v-bind:jobID="job.ID"></component>
</div>
</template>

<script>
import utils from './utils.js';
import JobConsoleProgress from './job-consoleprogress.vue';
import JobExport from './job-export.vue';
import JobPytorchTrain from './job-pytorch_train.vue';
import JobMultiExec from './job-multi_exec.vue';

export default {
	components: {
		'job-consoleprogress': JobConsoleProgress,
		'job-export': JobExport,
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

			this.$store.commit('setRouteData', {
				job: this.job,
			});
		});
	},
};
</script>
