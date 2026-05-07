INSERT INTO branches(id,name) VALUES ('11111111-1111-1111-1111-111111111111','Downtown Kitchen');
INSERT INTO ingredients(id,branch_id,name,unit,reorder_point) VALUES
('22222222-2222-2222-2222-222222222222','11111111-1111-1111-1111-111111111111','Chicken','kg',10),
('22222222-2222-2222-2222-222222222223','11111111-1111-1111-1111-111111111111','Rice','kg',8),
('22222222-2222-2222-2222-222222222224','11111111-1111-1111-1111-111111111111','Tomato','kg',5);
INSERT INTO suppliers(id,branch_id,name,email) VALUES ('33333333-3333-3333-3333-333333333333','11111111-1111-1111-1111-111111111111','Fresh Farm Supply','operations@example.test');
INSERT INTO menu_items(id,branch_id,name,price) VALUES
('44444444-4444-4444-4444-444444444444','11111111-1111-1111-1111-111111111111','Grilled chicken bowl',18),
('44444444-4444-4444-4444-444444444445','11111111-1111-1111-1111-111111111111','Tomato rice bowl',12);
INSERT INTO recipes VALUES
('44444444-4444-4444-4444-444444444444','22222222-2222-2222-2222-222222222222',0.25),
('44444444-4444-4444-4444-444444444444','22222222-2222-2222-2222-222222222223',0.15),
('44444444-4444-4444-4444-444444444445','22222222-2222-2222-2222-222222222223',0.2),
('44444444-4444-4444-4444-444444444445','22222222-2222-2222-2222-222222222224',0.1);
INSERT INTO inventory_movements(branch_id,ingredient_id,quantity,kind,reference)
SELECT branch_id,id,CASE name WHEN 'Chicken' THEN 5 WHEN 'Rice' THEN 40 ELSE 20 END,'opening','opening-count' FROM ingredients;
-- Synthetic training history, explicitly separated from the live stock ledger.
INSERT INTO demand_history(branch_id,ingredient_id,date,service,consumption)
SELECT i.branch_id,i.id,current_date - d,service,
round((CASE i.name WHEN 'Chicken' THEN 9 WHEN 'Rice' THEN 6 ELSE 3 END
 * CASE WHEN extract(isodow FROM current_date - d) IN (6,7) THEN 1.65 ELSE 1 END
 * CASE service WHEN 'dinner' THEN 1.3 ELSE 0.8 END
 * (0.92 + (d % 7) * 0.025))::numeric,3)
FROM ingredients i CROSS JOIN generate_series(1,84) d CROSS JOIN (VALUES('lunch'),('dinner')) s(service);
INSERT INTO kitchen_metrics(branch_id,prep_minutes,cancellation_rate,order_count,recorded_at)
SELECT '11111111-1111-1111-1111-111111111111',12 + d % 4,0.02,40, now() - d * interval '1 hour' FROM generate_series(1,48) d;
INSERT INTO knowledge(branch_id,source,content) VALUES
('11111111-1111-1111-1111-111111111111','SOP-DELIVERY-001','For a delivery shortage, reconcile purchase order quantity with signed receiving quantities. An external completed flag is not proof of full receipt. Check ingestion failures, waste and demand before assigning cause. Request replacement stock with manager approval. Record actual receipt before increasing stock.'),
('11111111-1111-1111-1111-111111111111','SOP-RECOVERY-001','Verify shortage recovery using recorded replacement receipt, positive ingredient balance and at least three new kitchen measurement windows totaling 30 orders over 10 minutes. Preparation time should be at most 18 minutes and cancellation rate at most 5 percent. Escalate if recovery does not occur within two hours.'),
('11111111-1111-1111-1111-111111111111','SOP-DEMAND-001','Compare demand against the same weekday and lunch or dinner service. Saturday and Sunday are naturally busier. Seasonal forecasts are planning estimates, not evidence that a delivery was fulfilled.');
