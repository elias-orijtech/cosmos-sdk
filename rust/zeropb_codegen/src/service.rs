use prost_types::{MethodDescriptorProto, ServiceDescriptorProto};
use crate::ctx::Context;
use std::fmt::Write;
use heck::ToSnakeCase;
use crate::r#type::gen_message_type;

pub(crate) fn gen_service(service: &ServiceDescriptorProto, ctx: &mut Context) -> anyhow::Result<()> {
    gen_service_server(service, ctx)?;
    gen_service_client(service, ctx)?;
    Ok(())
}

fn gen_service_server(service: &ServiceDescriptorProto, ctx: &mut Context) -> anyhow::Result<()> {
    write!(ctx, "trait {}Server<Ctx, Err> {{\n", service.name.clone().unwrap())?;
    for method in service.method.iter() {
        gen_server_method(method, ctx)?;
    }
    write!(ctx, "}}\n\n")?;
    Ok(())
}

pub(crate) fn gen_server_method(method: &MethodDescriptorProto, ctx: &mut Context) -> anyhow::Result<()> {
    let mut method_name = method.name.clone().unwrap();
    method_name = method_name.to_snake_case();
    let input_type = method.input_type.clone().unwrap();
    let output_type = method.output_type.clone().unwrap();
    write!(ctx, "    fn {}(&self, ctx: &Ctx, req: &", method_name)?;
    gen_message_type(&input_type, ctx)?;
    write!(ctx, ", resp: &mut ")?;
    gen_message_type(&output_type, ctx)?;
    write!(ctx, ") -> Result<(), Err>;\n")?;
    Ok(())
}

fn gen_service_client(service: &ServiceDescriptorProto, ctx: &mut Context) -> anyhow::Result<()> {
    write!(ctx, "struct {}Client<Client> {{\n", service.name.clone().unwrap())?;
    write!(ctx, "   client_conn: Client\n")?;
    write!(ctx, "}}\n\n")?;
    write!(ctx, "impl <Client> {}Client<Client> {{\n", service.name.clone().unwrap())?;
    for method in service.method.iter() {
        gen_client_method(method, ctx)?;
    }
    write!(ctx, "}}\n\n")?;
    Ok(())
}

pub(crate) fn gen_client_method(method: &MethodDescriptorProto, ctx: &mut Context) -> anyhow::Result<()> {
    let mut method_name = method.name.clone().unwrap();
    method_name = method_name.to_snake_case();
    let input_type = method.input_type.clone().unwrap();
    let output_type = method.output_type.clone().unwrap();
    write!(ctx, "    fn {}(&self, req: &", method_name)?;
    gen_message_type(&input_type, ctx)?;
    write!(ctx, ") -> Result<")?;
    gen_message_type(&output_type, ctx)?;
    write!(ctx, ", i32> {{\n")?;
    write!(ctx, "        unimplemented!()\n")?;
    write!(ctx, "    }}\n")?;
    Ok(())
}
