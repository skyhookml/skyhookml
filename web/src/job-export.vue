<template>
<div class="el-high">
	<template v-if="!job || !job.Done || job.Error">
		<job-consoleprogress :jobID="jobID"></job-consoleprogress>
	</template>
	<template v-else>
		<p>Export [{{ job.Name }}] completed successfully.</p>
		<a :href="job.Metadata" class="btn btn-primary">Download Export Zip Archive</a>
	</template>
</div>
</template>

<script>
import utils from './utils.js';
import JobConsoleProgress from './job-consoleprogress.vue';

export default {
	components: {
		'job-consoleprogress': JobConsoleProgress,
	},
	data: function() {
		return {
			job: null,
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
			utils.request(this, 'GET', '/jobs/'+this.jobID, null, (job) => {
				this.job = job;
			});
		},
	},
};
</script>
