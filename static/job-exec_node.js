import utils from './utils.js';
import JobConsole from './job-console.js';

export default {
	components: {
		'job-console': JobConsole,
	},
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
<div class="flex-container">
	<div class="mb-2">
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
	<div class="flex-content">
		<job-console :jobID="jobID" :lines="lines"></job-console>
	</div>
</div>
	`,
};
