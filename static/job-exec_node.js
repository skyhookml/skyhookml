import utils from './utils.js';

export default {
	data: function() {
		return {
			jobID: null,
			progress: 0,
			lines: [],
		};
	},
	created: function() {
		this.jobID = this.$route.params.jobid;
		this.fetch();
		this.interval = setInterval(this.fetch, 1000);
	},
	destroyed: function() {
		clearInterval(this.interval);
	},
	methods: {
		fetch: function() {
			utils.request(this, 'GET', '/jobs/'+this.jobID, null, (job) => {
				try {
					let s = JSON.parse(job.State);
					this.progress = parseInt(s.Progress);
					this.lines = s.Lines;
				} catch(e) {}
			});
		},
	},
	template: `
<div>
	<div>
		<div class="progress">
			<div
				class="progress-bar"
				role="progressbar"
				v-bind:style="{width: progress+'%'}"
				:aria-valuenow="progress"
				aria-valuemin="0"
				aria-valuemax="100"
				>
				{{ progress }}%
			</div>
		</div>
	</div>
	<div class="plaintext-div">
		<template v-for="line in lines">
			{{ line }}<br />
		</template>
	</div>
</div>
	`,
};
