package aci

import (
	"fmt"
	"log"

	"github.com/ciscoecosystem/aci-go-client/client"
	"github.com/ciscoecosystem/aci-go-client/models"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAciImportedContract() *schema.Resource {
	return &schema.Resource{
		Create: resourceAciImportedContractCreate,
		Update: resourceAciImportedContractUpdate,
		Read:   resourceAciImportedContractRead,
		Delete: resourceAciImportedContractDelete,

		Importer: &schema.ResourceImporter{
			State: resourceAciImportedContractImport,
		},

		SchemaVersion: 1,

		Schema: AppendBaseAttrSchema(map[string]*schema.Schema{
			"tenant_dn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"annotation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"name_alias": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"relation_vz_rs_if": &schema.Schema{
				Type: schema.TypeString,

				Optional: true,
			},
		}),
	}
}
func getRemoteImportedContract(client *client.Client, dn string) (*models.ImportedContract, error) {
	vzCPIfCont, err := client.Get(dn)
	if err != nil {
		return nil, err
	}

	vzCPIf := models.ImportedContractFromContainer(vzCPIfCont)

	if vzCPIf.DistinguishedName == "" {
		return nil, fmt.Errorf("ImportedContract %s not found", vzCPIf.DistinguishedName)
	}

	return vzCPIf, nil
}

func setImportedContractAttributes(vzCPIf *models.ImportedContract, d *schema.ResourceData) *schema.ResourceData {
	d.SetId(vzCPIf.DistinguishedName)
	d.Set("description", vzCPIf.Description)
	d.Set("tenant_dn", GetParentDn(vzCPIf.DistinguishedName))
	vzCPIfMap, _ := vzCPIf.ToMap()

	d.Set("name", vzCPIfMap["name"])

	d.Set("annotation", vzCPIfMap["annotation"])
	d.Set("name_alias", vzCPIfMap["nameAlias"])
	return d
}

func resourceAciImportedContractImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	log.Printf("[DEBUG] %s: Beginning Import", d.Id())
	aciClient := m.(*client.Client)

	dn := d.Id()

	vzCPIf, err := getRemoteImportedContract(aciClient, dn)

	if err != nil {
		return nil, err
	}
	schemaFilled := setImportedContractAttributes(vzCPIf, d)

	log.Printf("[DEBUG] %s: Import finished successfully", d.Id())

	return []*schema.ResourceData{schemaFilled}, nil
}

func resourceAciImportedContractCreate(d *schema.ResourceData, m interface{}) error {
	log.Printf("[DEBUG] ImportedContract: Beginning Creation")
	aciClient := m.(*client.Client)
	desc := d.Get("description").(string)

	name := d.Get("name").(string)

	TenantDn := d.Get("tenant_dn").(string)

	vzCPIfAttr := models.ImportedContractAttributes{}
	if Annotation, ok := d.GetOk("annotation"); ok {
		vzCPIfAttr.Annotation = Annotation.(string)
	}
	if NameAlias, ok := d.GetOk("name_alias"); ok {
		vzCPIfAttr.NameAlias = NameAlias.(string)
	}
	vzCPIf := models.NewImportedContract(fmt.Sprintf("cif-%s", name), TenantDn, desc, vzCPIfAttr)

	err := aciClient.Save(vzCPIf)
	if err != nil {
		return err
	}
	d.Partial(true)

	d.SetPartial("name")

	d.Partial(false)

	if relationTovzRsIf, ok := d.GetOk("relation_vz_rs_if"); ok {
		relationParam := relationTovzRsIf.(string)
		err = aciClient.CreateRelationvzRsIfFromImportedContract(vzCPIf.DistinguishedName, relationParam)
		if err != nil {
			return err
		}
		d.Partial(true)
		d.SetPartial("relation_vz_rs_if")
		d.Partial(false)

	}

	d.SetId(vzCPIf.DistinguishedName)
	log.Printf("[DEBUG] %s: Creation finished successfully", d.Id())

	return resourceAciImportedContractRead(d, m)
}

func resourceAciImportedContractUpdate(d *schema.ResourceData, m interface{}) error {
	log.Printf("[DEBUG] ImportedContract: Beginning Update")

	aciClient := m.(*client.Client)
	desc := d.Get("description").(string)

	name := d.Get("name").(string)

	TenantDn := d.Get("tenant_dn").(string)

	vzCPIfAttr := models.ImportedContractAttributes{}
	if Annotation, ok := d.GetOk("annotation"); ok {
		vzCPIfAttr.Annotation = Annotation.(string)
	}
	if NameAlias, ok := d.GetOk("name_alias"); ok {
		vzCPIfAttr.NameAlias = NameAlias.(string)
	}
	vzCPIf := models.NewImportedContract(fmt.Sprintf("cif-%s", name), TenantDn, desc, vzCPIfAttr)

	vzCPIf.Status = "modified"

	err := aciClient.Save(vzCPIf)

	if err != nil {
		return err
	}
	d.Partial(true)

	d.SetPartial("name")

	d.Partial(false)

	if d.HasChange("relation_vz_rs_if") {
		_, newRelParam := d.GetChange("relation_vz_rs_if")
		err = aciClient.DeleteRelationvzRsIfFromImportedContract(vzCPIf.DistinguishedName)
		if err != nil {
			return err
		}
		err = aciClient.CreateRelationvzRsIfFromImportedContract(vzCPIf.DistinguishedName, newRelParam.(string))
		if err != nil {
			return err
		}
		d.Partial(true)
		d.SetPartial("relation_vz_rs_if")
		d.Partial(false)

	}

	d.SetId(vzCPIf.DistinguishedName)
	log.Printf("[DEBUG] %s: Update finished successfully", d.Id())

	return resourceAciImportedContractRead(d, m)

}

func resourceAciImportedContractRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("[DEBUG] %s: Beginning Read", d.Id())

	aciClient := m.(*client.Client)

	dn := d.Id()
	vzCPIf, err := getRemoteImportedContract(aciClient, dn)

	if err != nil {
		d.SetId("")
		return nil
	}
	setImportedContractAttributes(vzCPIf, d)

	vzRsIfData, err := aciClient.ReadRelationvzRsIfFromImportedContract(dn)
	if err != nil {
		log.Printf("[DEBUG] Error while reading relation vzRsIf %v", err)

	} else {
		d.Set("relation_vz_rs_if", vzRsIfData)
	}

	log.Printf("[DEBUG] %s: Read finished successfully", d.Id())

	return nil
}

func resourceAciImportedContractDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("[DEBUG] %s: Beginning Destroy", d.Id())

	aciClient := m.(*client.Client)
	dn := d.Id()
	err := aciClient.DeleteByDn(dn, "vzCPIf")
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: Destroy finished successfully", d.Id())

	d.SetId("")
	return err
}
