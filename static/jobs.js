import utils from './utils.js';

const Jobs = {
	data: function() {
		return {
			jobs: [],
		};
	},
	created: function() {
		this.fetchJobs();
	},
	methods: {
		fetchJobs: function() {
			utils.request(this, 'GET', '/jobs', null, (data) => {
				this.jobs = data;
			});
		},
		selectJob: function(job) {
			this.$router.push('/ws/'+this.$route.params.ws+'/jobs/'+job.Op+'/'+job.ID);
		},
	},
	template: `
<div>
	<table class="table">
		<thead>
			<tr>
				<th>Name</th>
				<th>Type</th>
				<th>Time</th>
				<th>Status</th>
				<th></th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="job in jobs">
				<td>{{ job.Name }}</td>
				<td>{{ job.Type }}</td>
				<td>{{ job.StartTime }}</td>
				<td>
					<template v-if="!job.Done">
						Running
					</template>
					<template v-else-if="job.Error == ''">
						Success
					</template>
					<template v-else>
						Error: {{ job.Error }}
					</template>
				</td>
				<td>
					<button v-on:click="selectJob(job)" class="btn btn-primary">View</button>
				</td>
			</tr>
		</tbody>
	</table>
</div>
	`,
};
export default Jobs;
