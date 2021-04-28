<template>
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
	<div class="flex-content flex-container">
		<job-console :lines="lines"></job-console>
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
			progress: 0,
			lines: [],
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
				let progressState = JSON.parse(state.Datas['progress'])
				this.progress = parseInt(progressState);
				this.lines = state.Lines;
			});
		},
	},
};
</script>
