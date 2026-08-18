package geolocation
import("testing";"github.com/ArrowSK/ha-tkstar-gateway/internal/model")
func TestLearnResolve(t *testing.T){r:=New(t.TempDir(),"");good:=model.Position{Valid:true,Latitude:47.5,Longitude:19.05,Cells:[]model.Cell{{MCC:216,MNC:30,LAC:10,ID:20,Signal:50}}};r.Learn(good);p:=&model.Position{Cells:good.Cells};if !r.Resolve(p){t.Fatal("not resolved")};if p.Source!="LBS_LOCAL"||p.Latitude<47.49||p.Latitude>47.51{t.Fatalf("%+v",p)}}
