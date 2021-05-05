<template>
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
		<div class="d-flex">
			<form class="d-flex align-items-center" v-on:submit.prevent="downloadAs">
				<label class="mx-2 text-nowrap">Download as Format:</label>
				<select v-model="downloadFormat" class="form-select form-select-sm mx-2">
					<template v-if="item.Dataset.DataType == 'image'">
						<option value="png">PNG</option>
						<option value="jpeg">JPEG</option>
					</template>
					<template v-else-if="item.Dataset.DataType == 'table'">
						<option value="json">JSON</option>
						<option value="csv">CSV</option>
						<option value="sqlite3">SQLite3</option>
					</template>
					<template v-else>
						<option :value="item.Format">{{ item.Format }}</option>
					</template>
				</select>
				<button type="submit" class="btn btn-primary mx-2">Download</button>
			</form>
		</div>
	</template>
</div>
</template>

<script>
import utils from './utils.js';

export default {
	data: function() {
		return {
			item: null,
			metadata: {},

			loadedJSON: null,

			// for download as form, the format to download as
			downloadFormat: '',
		};
	},
	created: function() {
		const dsID = this.$route.params.dsid;
		const itemKey = this.$route.params.itemkey;
		utils.request(this, 'GET', '/datasets/'+dsID+'/items/'+itemKey, null, (item) => {
			this.item = item;
			this.downloadFormat = this.item.Format;

			let metadata = {};
			if(this.item.Dataset.Metadata) {
				for(let [k, v] of Object.entries(JSON.parse(this.item.Dataset.Metadata))) {
					metadata[k] = v;
				}
			}
			if(this.item.Metadata) {
				for(let [k, v] of Object.entries(JSON.parse(this.item.Metadata))) {
					metadata[k] = v;
				}
			}
			this.metadata = metadata;

			this.$store.commit('setRouteData', {
				dataset: this.item.Dataset,
				item: this.item,
			});
		});
	},
	methods: {
		loadJSON: function() {
			utils.request(this, 'GET', '/datasets/'+this.item.Dataset.ID+'/items/'+this.item.Key+'/get?format=json', null, (obj) => {
				this.loadedJSON = JSON.stringify(obj, null, 4);
			});
		},
		downloadAs: function() {
			window.location.href = '/datasets/'+this.item.Dataset.ID+'/items/'+this.item.Key+'/get?format='+this.downloadFormat;
		},
	},
};
</script>
