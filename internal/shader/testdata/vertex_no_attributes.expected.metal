struct Attributes {
};

struct Varyings {
	float4 Position [[position]];
	float4 M0;
};

vertex Varyings Vertex(
	uint vid [[vertex_id]],
	const device Attributes* attributes [[buffer(0)]]) {
	Varyings varyings = {};
	varyings.Position = float4(0.0, 0.0, 0.0, 1.0);
	varyings.M0 = float4(1.0);
	return varyings;
}

fragment float4 Fragment(
	Varyings varyings [[stage_in]],
	bool front_facing [[front_facing]]) {
	return varyings.M0;
}
