import utils from './utils.js';

export default {
	data: function() {
		return {
			datasets: [],
			addForm: {
				// the selected tool ID and actual object
				// object is filled in by changedTool
				tool: null,
				toolObj: null,

				// name and type of dataset in case datasetMode=='new'
				// if toolObj.DataType is set (only one option), then datasetType is not configurable
				datasetName: '',
				datasetType: '',

				// input dataset IDs to use for annotation
				// corresponds to toolObj.Inputs
				inputIDs: [],
			},

			tools: {
				"shape": {
					ID: "shape",
					Name: "Object Detection and Segmentation",
					Help: "Annotate bounding boxes, polygons, lines, and other shapes. Each shape can be further labeled with a category and a track ID.",
					Inputs: [{
						Name: "Image/Video",
						DataTypes: ["image", "video"],
						Help: "Select an image or video dataset to label. If you have not imported the data yet, first head to Dashboard and then Quickstart Import.",
					}],
					DataTypes: [
						{ID: "detection", Label: "Object Detections. Choose this type if you are annotating bounding boxes for object detection."},
						{ID: "shape", Label: "Shapes. Choose this type if you are annotating shapes other than bounding boxes, e.g., polygons for image segmentation."},
					],
				},
				"int": {
					ID: "int",
					Name: "Image Classification or Regression",
					Help: "Annotate integers, such as category IDs for image classification or arbitrary numbers for regression.",
					Inputs: [{
						Name: "Image/Video",
						DataTypes: ["image", "video"],
						Help: "Select an image or video dataset to label. If you have not imported the data yet, first head to Dashboard and then Quickstart Import.",
					}],
					DataType: "int",
				},
				"detection-to-track": {
					ID: "detection-to-track",
					Name: "Group Detections into Tracks",
					Help: "Given video, along with object detections computed or previously annotated in the video, group together detections into tracks to derive training data for an object tracking model.",
					Inputs: [{
						Name: "Video",
						DataTypes: ["video"],
						Help: "Select a video dataset to label. If you have not imported the data yet, first head to Dashboard and then Quickstart Import.",
					}, {
						Name: "Detections",
						DataTypes: ["detection"],
						Help: "Select a dataset of object detections corresponding to the video.",
					}],
					DataType: "detection",
				},
			},
		};
	},
	created: function() {
		utils.request(this, 'GET', '/datasets', null, (data) => {
			this.datasets = data;
		});
	},
	methods: {
		selectTool: function(tool) {
			// update the cached addForm.toolObj
			// if tool has a single DataType, we set datasetType
			this.addForm.tool = tool;
			let toolObj = this.tools[this.addForm.tool];
			this.addForm.toolObj = toolObj;
			this.addForm.inputIDs = [];
			for(let i = 0; i < toolObj.Inputs.length; i++) {
				this.addForm.inputIDs.push(null);
			}
			this.addForm.datasetType = '';
			if(toolObj.DataType) {
				this.addForm.datasetType = toolObj.DataType;
			}
		},
		addAnnoset: function() {
			let handle = async () => {
				let dataset;
				try {
					let params = {
						name: this.addForm.datasetName,
						data_type: this.addForm.datasetType,
					};
					dataset = await utils.request(this, 'POST', '/datasets', params);
				} catch(e) {
					return;
				}

				let annoset;
				try {
					let params = {
						ds_id: dataset.ID,
						inputs: this.addForm.inputIDs.join(','),
						tool: this.addForm.tool,
						params: '',
					};
					annoset = await utils.request(this, 'POST', '/annotate-datasets', params);
				} catch(e) {
					return;
				}
				this.$router.push('/ws/'+this.$route.params.ws+'/annotate/'+annoset.Tool+'/'+annoset.ID);
			};
			handle();
		},
	},
	template: `
<div class="small-container">
	<h3>Add Annotation Dataset</h3>
	<template v-if="!addForm.tool">
		<p>Select an annotation tool:</p>
		<template v-for="tool in tools">
			<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectTool(tool.ID)">
				<div class="card-body">
					<h5 class="card-title">{{ tool.Name }}</h5>
					<p class="card-text">{{ tool.Help }}</p>
				</div>
			</div>
		</template>
	</template>
	<template v-else>
		<form v-on:submit.prevent="addAnnoset">
			<div class="row mb-2">
				<label class="col-sm-4 col-form-label">Name</label>
				<div class="col-sm-8">
					<input v-model="addForm.datasetName" type="text" class="form-control">
					<small class="form-text text-muted">A name for these annotations.</small>
				</div>
			</div>
			<div class="row mb-2" v-if="addForm.toolObj">
				<label class="col-sm-4 col-form-label">Data Type</label>
				<div class="col-sm-8">
					<template v-if="!addForm.toolObj.DataType">
						<div class="form-check" v-for="dt in addForm.toolObj.DataTypes">
							<input class="form-check-input" type="radio" v-model="addForm.datasetType" :value="dt.ID">
							<label class="form-check-label">{{ dt.Label }}</label>
						</div>
					</template>
					<template v-else>
						<input type="text" readonly class="form-control-plaintext" :value="addForm.datasetType" />
					</template>
				</div>
			</div>
			<template v-for="(input, i) in addForm.toolObj.Inputs">
				<div class="row mb-2">
					<label class="col-sm-4 col-form-label">Input {{ input.Name }}</label>
					<div class="col-sm-8">
						<select v-model="addForm.inputIDs[i]" class="form-select">
							<template v-for="ds in datasets">
								<!-- Only show datasets that match the type of this input. -->
								<option
									v-if="!input.DataTypes || input.DataTypes.includes(ds.DataType)"
									:key="ds.ID"
									:value="ds.ID">
									{{ ds.Name }}
								</option>
							</template>
						</select>
						<small v-if="input.Help" class="form-text text-muted">{{ input.Help }}</small>
					</div>
				</div>
			</template>
			<div class="row mb-2">
				<div class="col-sm-12">
					<button type="submit" class="btn btn-primary">Add Annotation Dataset</button>
				</div>
			</div>
		</form>
	</template>
</div>
	`,
};
