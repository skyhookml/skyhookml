import utils from './utils.js';

export default {
	data: function() {
		return {
			item: null,
			metadata: {},

			loadedJSON: null,
		};
	},
	created: function() {
		const dsID = this.$route.params.dsid;
		const itemKey = this.$route.params.itemkey;
		utils.request(this, 'GET', '/datasets/'+dsID+'/items/'+itemKey, null, (item) => {
			this.item = item;
			try {
				this.metadata = JSON.parse(this.item.Metadata);
			} catch(e) {}
		});
	},
	methods: {
		loadJSON: function() {
			utils.request(this, 'GET', '/datasets/'+this.item.Dataset.ID+'/items/'+this.item.Key+'/get?format=json', null, (obj) => {
				this.loadedJSON = JSON.stringify(obj, null, 4);
			});
		},
	},
	template: `
<div>
	<template v-if="item">
		<table class="table table-sm">
			<tbody>
				<tr>
					<th>Dataset Name</th>
					<td>{{ item.Dataset.Name }}</td>
				</tr>
				<tr>
					<th>Data Type</th>
					<td>{{ item.Dataset.DataType }}</td>
				</tr>
				<tr>
					<th>Key</th>
					<td>{{ item.Key }}</td>
				</tr>
			</tbody>
		</table>
		<template v-if="item.Dataset.DataType == 'video'">
			<h4>Metadata</h4>
			<table class="table table-sm">
				<tbody>
					<tr>
						<th>Dimensions</th>
						<td>{{ metadata.Dims[0] }}x{{ metadata.Dims[1] }}</td>
					</tr>
					<tr>
						<th>Framerate</th>
						<td>{{ metadata.Framerate[0] }}/{{ metadata.Framerate[1] }}</td>
					</tr>
					<tr>
						<th>Duration</th>
						<td>{{ metadata.Duration }}</td>
					</tr>
				</tbody>
			</table>
			<h4>Video</h4>
			<video controls :src="'/datasets/'+item.Dataset.ID+'/items/'+item.Key+'/get?format=mp4'" class="explore-result-img"></video>
		</template>
		<template v-else-if="item.Dataset.DataType == 'image'">
			<img :src="'/datasets/'+item.Dataset.ID+'/items/'+item.Key+'/get?format=jpeg'" class="explore-result-img" />
		</template>
		<template v-else-if="item.Dataset.DataType == 'file'">
			<h4>File: {{ metadata.Filename }}</h4>
			<a :href="'/datasets/'+item.Dataset.ID+'/items/'+item.Key+'/get?format=file'" class="btn btn-primary">Download</a>
		</template>
		<template v-else-if="item.Format == 'json' || item.Ext == 'json'">
			<template v-if="Object.keys(metadata).length > 0">
				<h4>Metadata</h4>
				<table class="table table-sm">
					<tbody>
						<tr v-for="(value, name) in metadata">
							<th>{{ name }}</th>
							<td>{{ value }}</td>
						</tr>
					</tbody>
				</table>
			</template>
			<h4>JSON</h4>
			<template v-if="loadedJSON">
				<pre>{{ loadedJSON }}</pre>
			</template>
			<template v-else>
				<button class="btn btn-primary" v-on:click="loadJSON">Load JSON</button>
			</template>
		</template>
	</template>
</div>
	`,
};
